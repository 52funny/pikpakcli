package find

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/spf13/cobra"
)

// Register the command
var FindCmd = &cobra.Command{
	Use:     "find",
	Aliases: []string{"f"},
	Short:   "Find files on PikPak",
	Long:    `Find files and folders on PikPak using keywords`,
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			return
		}
		handleFind(cmd, &p, args)
	},
}

// Flags
var (
	limit         int
	humanFlag     bool
	detailFlag    bool
	pathFlag      string
	recursiveFlag bool
)

func init() {
	// Define flags
	// Limit number of results
	FindCmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of results to return")
	// Human readable file sizes
	FindCmd.Flags().BoolVarP(&humanFlag, "human", "H", false, "Print human readable file sizes")
	// Return in long format
	FindCmd.Flags().BoolVarP(&detailFlag, "detail", "d", false, "Print detailed information (ID, size, modified time)")
	// Search path
	FindCmd.Flags().StringVarP(&pathFlag, "path", "p", "/", "Search path (default: /)")
	// Recursive search
	FindCmd.Flags().BoolVarP(&recursiveFlag, "recursive", "r", false, "Recursively search in subdirectories")
}

func handleFind(cmd *cobra.Command, p *api.PikPak, args []string) {
	var phrase, searchPath string

	// Parse arguments
	if len(args) == 0 {
		fmt.Println("Error: search phrase is required")
		cmd.Usage()
		return
	} else if len(args) == 1 {
		// Only one argument: it's the phrase
		phrase = args[0]
		searchPath = pathFlag
	} else if len(args) == 2 {
		// Two arguments: first is path, second is phrase
		searchPath = args[0]
		phrase = args[1]
	} else {
		fmt.Println("Error: too many arguments")
		cmd.Usage()
		return
	}

	// Normalize path: ensure starts with /, remove trailing /
	if !strings.HasPrefix(searchPath, "/") {
		searchPath = "/" + searchPath
	}
	// Remove trailing slash (except for root)
	if searchPath != "/" && strings.HasSuffix(searchPath, "/") {
		searchPath = strings.TrimSuffix(searchPath, "/")
	}

	var results []api.SearchResult
	var err error

	// Use recursive or non-recursive search based on flag
	if recursiveFlag {
		results, err = p.SearchFiles(phrase, limit)
	} else {
		results, err = p.SearchFilesInPath(phrase, searchPath, limit)
	}

	if err != nil {
		fmt.Printf("Find failed: %v\n", err)
		return
	}

	// Filter results by search path (only needed for recursive search)
	var filteredResults []api.SearchResult
	if recursiveFlag {
		for _, result := range results {
			// Check if result is under the search path
			var isUnderPath bool
			if searchPath == "/" {
				// All results are under root
				isUnderPath = true
			} else if result.Path == searchPath {
				// Exact match
				isUnderPath = true
			} else if strings.HasPrefix(result.Path, searchPath+"/") {
				// Under the search path
				isUnderPath = true
			}
			if isUnderPath {
				filteredResults = append(filteredResults, result)
			}
		}
	} else {
		// Non-recursive search already filters by path
		filteredResults = results
	}

	if len(filteredResults) == 0 {
		fmt.Printf("No files found for phrase: %s in path: %s\n", phrase, searchPath)
		return
	}

	if !detailFlag {
		fmt.Printf("Find results for '%s' in '%s' (%d results):\n\n", phrase, searchPath, len(filteredResults))
	}

	for _, result := range filteredResults {
		if detailFlag {
			size := result.Size
			if humanFlag {
				// Convert size to int64 for formatting
				if sizeInt, err := strconv.ParseInt(result.Size, 10, 64); err == nil {
					size = formatSize(sizeInt)
				}
			}

			fmt.Printf("%s %s", result.ID, size)
			if result.Kind == "drive#file" {
				fmt.Printf(" %s %s", result.ModifiedTime.Format("2006-01-02 15:04:05"), result.Path)
			} else {
				fmt.Printf(" %s %s (folder)", result.ModifiedTime.Format("2006-01-02 15:04:05"), result.Path)
			}
		} else {
			fmt.Print(result.Path)
		}
		fmt.Printf("\n")
	}
}

// Human readable file size formatting
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
