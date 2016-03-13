package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Get list of files below a directory.
// Returns filenames only.
// Strips the root directory name form the filename.
// Recursively visits all files below this level.
func FilesInDirRecursive(root string) ([]string, error) {
	found := []string{}

	fi, err := os.Stat(root)
	switch {
	case err != nil:
		return found, err
	case fi.IsDir():
		if strings.HasSuffix(root, "/") == false {
			root = root + "/"
		}
	default:
		return found, fmt.Errorf("%v: Not a directory", root)
	}

	// Create a callback function
	walker := func(path string, info os.FileInfo, err error) error {

		fi2, err := os.Stat(path)
		switch {
		case err != nil:
			return nil
		case fi2.IsDir():
			return nil
		default:
			p := path[len(root):]
			found = append(found, p)
			return nil
		}
	}

	// Start walking!
	filepath.Walk(root, walker)
	return found, nil

}

// Get list of files below a directory, without recursion
func FilesInDirNotRecursive(root string) ([]string, error) {
	found := []string{}

	fi, err := os.Stat(root)
	switch {
	case err != nil:
		return found, err
	case fi.IsDir():
		if strings.HasSuffix(root, "/") == false {
			root = root + "/"
		}
	default:
		return found, fmt.Errorf("%v: Not a directory", root)
	}

	// Create a callback function
	walker := func(path string, info os.FileInfo, err error) error {

		fi2, err := os.Stat(path)
		switch {
		case err != nil:
			return nil
		case fi2.IsDir():
			if path == root {
				return nil
			}
			return filepath.SkipDir
		default:
			p := path[len(root):]
			found = append(found, p)
			return nil
		}
	}

	// Start walking!
	filepath.Walk(root, walker)
	return found, nil

}
