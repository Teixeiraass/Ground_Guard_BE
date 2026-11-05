package util

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"
	"os"
	"path/filepath"
)

const maxImageDimension = 4096

var ErrInvalidImage = errors.New("arquivo não é uma imagem PNG ou JPEG válida")

// SaveUserImage validates an uploaded picture by its content (not its name),
// rejects decompression bombs, and re-encodes it as JPEG. Re-encoding drops
// EXIF metadata such as the GPS position of the photo. The file name is
// random so it can't be guessed or used for path traversal.
func SaveUserImage(imageData []byte, _ string) (string, error) {
	switch http.DetectContentType(imageData) {
	case "image/png", "image/jpeg":
	default:
		return "", ErrInvalidImage
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 ||
		cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
		return "", ErrInvalidImage
	}
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", ErrInvalidImage
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}

	dir := "uploads/profile"
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("erro ao criar diretório de profile: %w", err)
	}

	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, "profile_"+hex.EncodeToString(name)+".jpg")
	if err := os.WriteFile(filePath, out.Bytes(), 0o640); err != nil {
		return "", err
	}
	return filePath, nil
}
