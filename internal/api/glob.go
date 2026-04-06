package api

import (
	"fmt"
	"path"
	"strings"
)

type remotePatternProvider interface {
	GetPathFolderId(dirPath string) (string, error)
	GetFolderFileStatList(parentId string) ([]FileStat, error)
}

func ExpandRemotePatterns(p remotePatternProvider, basePath string, patterns []string, keepRelative bool) ([]string, error) {
	expanded := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		matches, err := expandRemotePattern(p, basePath, pattern, keepRelative)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, matches...)
	}
	return expanded, nil
}

func expandRemotePattern(p remotePatternProvider, basePath string, pattern string, keepRelative bool) ([]string, error) {
	if !hasRemoteWildcard(pattern) {
		return []string{pattern}, nil
	}

	resolvedPattern := path.Clean(pattern)
	if !path.IsAbs(resolvedPattern) {
		resolvedPattern = path.Clean(path.Join("/", basePath, pattern))
	}

	parentPath := path.Dir(resolvedPattern)
	if parentPath == "." {
		parentPath = "/"
	}

	parentID := ""
	var err error
	if parentPath != "/" {
		parentID, err = p.GetPathFolderId(parentPath)
		if err != nil {
			return nil, err
		}
	}

	files, err := p.GetFolderFileStatList(parentID)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0)
	namePattern := path.Base(resolvedPattern)
	for _, file := range files {
		matched, err := path.Match(namePattern, file.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid wildcard pattern %s: %w", pattern, err)
		}
		if !matched {
			continue
		}

		matchPath := path.Join(parentPath, file.Name)
		if keepRelative && !path.IsAbs(pattern) {
			matches = append(matches, relativeRemotePath(basePath, matchPath))
			continue
		}
		matches = append(matches, matchPath)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches found for %s", pattern)
	}

	return matches, nil
}

func hasRemoteWildcard(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func relativeRemotePath(basePath string, fullPath string) string {
	base := path.Clean(basePath)
	full := path.Clean(fullPath)
	if base == "." || base == "" || base == "/" {
		return strings.TrimPrefix(full, "/")
	}
	prefix := base + "/"
	if strings.HasPrefix(full, prefix) {
		return strings.TrimPrefix(full, prefix)
	}
	return full
}
