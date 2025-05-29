package files

import (
	"os"
	"path/filepath"
)

func Write(n, c string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, n+".tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	if _, err := f.WriteString(c); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	err = rename(f.Name(), filepath.Join(dir, n))
	return err
}

func Read(n string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(dir, n))
	return string(data), err
}
