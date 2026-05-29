package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateQRToken(length int) (string, error) {
	bytes := make([]byte, length)
	
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("erro ao gerar bytes aleatórios: %v", err)
	}
	
	return hex.EncodeToString(bytes), nil
}