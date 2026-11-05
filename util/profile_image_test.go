package util

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func pngBytes(t *testing.T, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func inTempDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestSaveUserImageReencodesWithRandomName(t *testing.T) {
	inTempDir(t)

	path1, err := SaveUserImage(pngBytes(t, 10, 10), "../../etc/passwd")
	require.NoError(t, err)
	path2, err := SaveUserImage(pngBytes(t, 10, 10), "same-user")
	require.NoError(t, err)

	require.NotEqual(t, path1, path2)
	require.Equal(t, filepath.Join("uploads", "profile"), filepath.Dir(path1), "username must not influence the path")
	require.True(t, strings.HasSuffix(path1, ".jpg"))

	stored, err := os.ReadFile(path1)
	require.NoError(t, err)
	_, format, err := image.Decode(bytes.NewReader(stored))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
}

func TestSaveUserImageRejectsNonImagesAndBombs(t *testing.T) {
	inTempDir(t)

	for name, data := range map[string][]byte{
		"html disguised":   []byte("<html><script>alert(1)</script></html>"),
		"empty":            {},
		"truncated png":    pngBytes(t, 10, 10)[:30],
		"too large (bomb)": pngBytes(t, maxImageDimension+1, 1),
	} {
		_, err := SaveUserImage(data, "u")
		require.ErrorIs(t, err, ErrInvalidImage, name)
	}
}
