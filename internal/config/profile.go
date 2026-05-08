package config

import (
	"os"
	"path/filepath"
)

func DefaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".base-node-helper"), nil
}
