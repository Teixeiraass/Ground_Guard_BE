package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skip2/go-qrcode"
)

func GenerateQRCodeImage(token string) (string, error) {
	dir := "uploads/qrcodes"
	
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("erro ao criar diretório de qrcode: %v", err)
	}

	filename := fmt.Sprintf("qr_%s.png", token)
	filePath := filepath.Join(dir, filename)

	err := qrcode.WriteFile(token, qrcode.Medium, 256, filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao escrever arquivo do qrcode: %v", err)
	}

	return filename, nil
}