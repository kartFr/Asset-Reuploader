package files

import (
	"os"
)

func Write(n, c string) error {
	f, err := os.OpenFile(n, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(c)
	return err
}

func Read(n string) (string, error) {
	data, err := os.ReadFile(n)
	return string(data), err
}
