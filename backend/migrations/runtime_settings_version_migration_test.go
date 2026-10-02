package migrations

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeSettingsVersionMigrationSeedsCASRowIdempotently(t *testing.T) {
	sql, err := os.ReadFile("241_runtime_settings_version.sql")
	require.NoError(t, err)

	content := string(sql)
	require.Contains(t, content, "__sub2api_runtime_settings_version")
	require.Contains(t, content, "ON CONFLICT (key) DO NOTHING")
}
