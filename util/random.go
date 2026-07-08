package util

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomInt generate a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1) // 0->max-min
}

// RandomString generate a random string of lenght n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

func RandomUser() string {
	return RandomString(6)
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}

func RandomFirwareVersion() string {
	return fmt.Sprintf(
		"%d.%d.%d",
		RandomInt(0, 9),
		RandomInt(0, 9),
		RandomInt(0, 99),
	)
}

func RandomStatus() string {
	statuses := []string{"ativo", "inativo"}

	return statuses[rand.Intn(len(statuses))]
}

func RandomUuid() uuid.UUID {
	return uuid.New()
}

func RandomImage() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rand.Intn(256)),
				G: uint8(rand.Intn(256)),
				B: uint8(rand.Intn(256)),
				A: 255,
			})
		}
	}

	var buffer bytes.Buffer

	if err := png.Encode(&buffer, img); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func RandomIrrigationMode() string {
	irrigationMode := []string{"MANUAL", "INTELIGENTE"}

	return irrigationMode[rand.Intn(len(irrigationMode))]
}
