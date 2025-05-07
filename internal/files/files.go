package files

import (
	"os"
	"path/filepath"
	"strings"
)

func getDir() string {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		if !strings.HasPrefix(exeDir, os.TempDir()) {
			return exeDir
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func Write(n, c string) error {
	dir := getDir()
	f, err := os.OpenFile(filepath.Join(dir, n), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(c)
	return err
}

func Read(n string) (string, error) {
	dir := getDir()
	data, err := os.ReadFile(filepath.Join(dir, n))
	return string(data), err
}
