package utils

import (
	"fmt"
	"path/filepath"
)

func GetEmbedBinName(name string) string {
	if len(name) == 0 {
		return "_embed"
	}
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	fmt.Println(base, ext)
	if len(ext) > 0 {
		base = base[:len(base)-len(ext)]
	}
	return base + "_embed" + ext
}
