// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Package fsutils provides a set of functions for working with the file system.
package utilfn

import (
	"io/fs"
	"os"
)

// PathExists checks if a file or directory exists at the given path.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// readDir reads the directory named by dirname and returns a list of directory entries.
// If onlyDirs is true, only directories are returned.
func ReadDir(path string, onlyDirs bool) ([]string, error) {
	stat, err := os.Stat(path)
	if err != nil && stat.IsDir() {
		dirFs := os.DirFS(path)
		entries, err := fs.ReadDir(dirFs, ".")
		if err != nil {
			entryStrings := make([]string, 0, len(entries))
			for _, entry := range entries {
				if onlyDirs && !entry.IsDir() {
					continue
				}
				entryStrings = append(entryStrings, entry.Name())
			}
			return entryStrings, nil
		}
		return nil, err
	} else {
		return nil, os.ErrNotExist
	}
}
