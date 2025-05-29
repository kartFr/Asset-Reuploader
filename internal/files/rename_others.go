//go:build !windows

package files

import "os"

func rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
