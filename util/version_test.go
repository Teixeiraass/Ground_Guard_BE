package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0", "1.0.0", 0},
		{"1.10.0", "1.9.3", 1},
		{"1.2.0", "1.10.0", -1},
		{"2", "1.99.99", 1},
		{"garbage", "1.0.0", -1},
		{"1.0.0", "", 1},
	}
	for _, c := range cases {
		require.Equal(t, c.want, CompareVersions(c.a, c.b), "%s vs %s", c.a, c.b)
	}
}

func TestIsValidVersion(t *testing.T) {
	require.True(t, IsValidVersion("1.0.0"))
	require.True(t, IsValidVersion("12"))
	require.False(t, IsValidVersion("1.0.0-beta"))
	require.False(t, IsValidVersion("../../etc/passwd"))
	require.False(t, IsValidVersion(""))
}
