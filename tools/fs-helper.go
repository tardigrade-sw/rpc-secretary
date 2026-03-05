package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

func GetFilesByType(root, ext string) ([]string, error) {
	var files []string

	// Ensure the extension starts with a dot for consistent matching
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ext {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
