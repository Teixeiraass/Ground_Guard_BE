package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func SaveUserImage(imageData []byte, username string) (string, error) {
	dir := "uploads/profile"

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf(
			"erro ao criar diretório de profile: %w",
			err,
		)
	}

	filename := fmt.Sprintf(
		"profile_%s.png",
		username,
	)

	filePath := filepath.Join(
		dir,
		filename,
	)

	err := os.WriteFile(
		filePath,
		imageData,
		0644,
	)

	if err != nil {
		return "", err
	}

	return filePath, nil
}
