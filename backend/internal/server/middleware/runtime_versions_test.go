package middleware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeRuntimePriceVersionTracksAllPricingFiles(t *testing.T) {
	dataDir := t.TempDir()
	fallbackFile := filepath.Join(t.TempDir(), "fallback.json")
	overrideFile := filepath.Join(t.TempDir(), "override.json")
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "model_pricing.json"), []byte(`{"model":{"input":1}}`), 0o600))
	require.NoError(t, os.WriteFile(fallbackFile, []byte(`{"fallback":1}`), 0o600))
	require.NoError(t, os.WriteFile(overrideFile, []byte(`{"override":1}`), 0o600))

	before := computeRuntimePriceVersion([]string{dataDir, fallbackFile, overrideFile})
	require.Len(t, before, 16)
	require.NotEqual(t, "fallback", before)

	require.NoError(t, os.WriteFile(overrideFile, []byte(`{"override":2}`), 0o600))
	after := computeRuntimePriceVersion([]string{dataDir, fallbackFile, overrideFile})
	require.NotEqual(t, before, after)
}

func TestComputeRuntimePriceVersionFallsBackWithoutFiles(t *testing.T) {
	require.Equal(t, "fallback", computeRuntimePriceVersion([]string{t.TempDir()}))
}
