package files

import (
	"os"
	"path/filepath"
)

func getDir() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(path)
	return dir, nil
}

func Write(n, c string) error {
	dir, err := getDir()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(dir, n), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(c)
	return err
}

func Read(n string) (string, error) {
	dir, err := getDir()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(dir, n))
	return string(data), err
}
