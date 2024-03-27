package metrics

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseMetricsFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		setupTestData     func(t *testing.T, tmpDir, metricsFile string)                      // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir, metricsFile string, parsedMetrics *File) // Post function validation/cleanup
		wantErr           bool
	}{
		{
			name: "non_existing_directory",
			setupTestData: func(t *testing.T, tmpDir, metricsFile string) {
				t.Helper()
				// make directory absent
				_ = os.RemoveAll(tmpDir)
			},
			postCheckTestData: func(t *testing.T, tmpDir, metricsFile string, parsedMetrics *File) {
				t.Helper()
				// directory shall not be created
				_, err := os.Stat(tmpDir)
				require.Error(t, err, os.ErrNotExist)
				require.Nil(t, parsedMetrics)
			},
			wantErr: true,
		},
		{
			name: "empty_directory",
			setupTestData: func(t *testing.T, tmpDir, metricsFile string) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, metricsFile string, parsedMetrics *File) {
				t.Helper()

				// directory shall not be removed
				_, err := os.Stat(tmpDir)
				require.NoError(t, err)

				// file shall not be created
				_, err = os.Stat(filepath.Clean(filepath.Join(tmpDir, metricsFile)))
				require.Error(t, err, os.ErrNotExist)
				require.Nil(t, parsedMetrics)
			},
			wantErr: true,
		},
		{
			name: "empty_file",
			setupTestData: func(t *testing.T, tmpDir, metricsFile string) {
				t.Helper()
				err := os.WriteFile(filepath.Join(tmpDir, metricsFile), []byte(""), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, metricsFile string, parsedMetrics *File) {
				t.Helper()
				// directory shall not be removed
				_, err := os.Stat(tmpDir)
				require.NoError(t, err)

				// file shall not be removed
				_, err = os.Stat(filepath.Clean(filepath.Join(tmpDir, metricsFile)))
				require.NoError(t, err)
				require.Nil(t, parsedMetrics)
			},
			wantErr: true,
		},
		{
			name: "corrupted_file",
			setupTestData: func(t *testing.T, tmpDir, metricsFile string) {
				t.Helper()
				fileContent := `{
"db_instance_id": "1bed5f0d-cc3a-11ee-bd8a-c84bd64e0288",
"pillar_version": "8.0.35-27-debug",
}
`
				err := os.WriteFile(filepath.Join(tmpDir, metricsFile), []byte(fileContent), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, metricsFile string, parsedMetrics *File) {
				t.Helper()
				// directory shall not be removed
				_, err := os.Stat(tmpDir)
				require.NoError(t, err)

				// file shall not be removed
				_, err = os.Stat(filepath.Clean(filepath.Join(tmpDir, metricsFile)))
				require.NoError(t, err)
				require.Nil(t, parsedMetrics)
			},
			wantErr: true,
		},
		{
			name: "valid_file",
			setupTestData: func(t *testing.T, tmpDir, metricsFile string) {
				t.Helper()
				fileContent := `{
"db_instance_id": "1bed5f0d-cc3a-11ee-bd8a-c84bd64e0288",
"pillar_version": "8.0.35-27-debug",
"active_plugins": [
    "keyring_file",
    "binlog",
    "mysqlx",
    "group_replication"
],
"active_components": [
    "file://component_percona_telemetry"
],
"uptime": "112",
"boolean_value_true": true,
"boolean_value_false": false,
"boolean_value_true_string": "true",
"boolean_value_false_string": "false",
"boolean_value_1": "1",
"boolean_value_0": "0",
"databases_count": "0",
"databases_size": "0",
"se_engines_in_use": []
}
`
				err := os.WriteFile(filepath.Join(tmpDir, metricsFile), []byte(fileContent), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, metricsFile string, parsedMetrics *File) {
				t.Helper()
				// directory shall not be removed
				_, err := os.Stat(tmpDir)
				require.NoError(t, err)

				// file shall not be removed
				_, err = os.Stat(filepath.Clean(filepath.Join(tmpDir, metricsFile)))
				require.NoError(t, err)
				require.NotNil(t, parsedMetrics)
				require.Equal(t, "1bed5f0d-cc3a-11ee-bd8a-c84bd64e0288", parsedMetrics.Metrics["db_instance_id"])
				require.Equal(t, "8.0.35-27-debug", parsedMetrics.Metrics["pillar_version"])
				require.Equal(t, "112", parsedMetrics.Metrics["uptime"])
				require.Equal(t, "0", parsedMetrics.Metrics["boolean_value_false"])
				require.Equal(t, "0", parsedMetrics.Metrics["boolean_value_0"])
				require.Equal(t, "1", parsedMetrics.Metrics["boolean_value_true"])
				require.Equal(t, "1", parsedMetrics.Metrics["boolean_value_1"])
				require.Equal(t, "0", parsedMetrics.Metrics["boolean_value_false_string"])
				require.Equal(t, "1", parsedMetrics.Metrics["boolean_value_true_string"])
				require.Equal(t, "[]", parsedMetrics.Metrics["se_engines_in_use"])
				require.Equal(t, "[\"file://component_percona_telemetry\"]", parsedMetrics.Metrics["active_components"])
			},
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, err := os.MkdirTemp("", "test-metrics")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			currTime := time.Now()
			token := uuid.New().String()
			metricsFile := fmt.Sprintf("%d-%s.json", currTime.Unix(), token)
			tt.setupTestData(t, tmpDir, metricsFile)

			f, err := parseMetricsFile(filepath.Join(tmpDir, metricsFile))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, f)
			}
			tt.postCheckTestData(t, tmpDir, metricsFile, f)
		})
	}
}
