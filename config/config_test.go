// Copyright (C) 2024 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package config

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestInitConfig(t *testing.T) { //nolint:paralleltest
	testCases := []struct {
		name           string
		setupTestData  func(t *testing.T)
		expectedConfig Config
	}{
		{
			name: "all_default_values",
			setupTestData: func(t *testing.T) {
				t.Helper()
			},
			expectedConfig: Config{
				PSMetricsPath:                filepath.Join("/usr", "local", "percona", "telemetry", "ps"),
				PSMDBMetricsPath:             filepath.Join("/usr", "local", "percona", "telemetry", "psmdb"),
				PXCMetricsPath:               filepath.Join("/usr", "local", "percona", "telemetry", "pxc"),
				PGMetricsPath:                filepath.Join("/usr", "local", "percona", "telemetry", "pg"),
				TelemetryCheckInterval:       telemetryCheckIntervalDefault,
				TelemetryResendTimeout:       telemetryResendIntervalDefault,
				TelemetryHistoryPath:         filepath.Join("/usr", "local", "percona", "telemetry", "history"),
				TelemetryHistoryKeepInterval: historyKeepIntervalDefault,
				PerconaTelemetryURL:          perconaTelemetryURLDefault,
			},
		},
		{
			name: "redefine_all_values",
			setupTestData: func(t *testing.T) {
				t.Helper()

				t.Setenv(telemetryRootPath, "/tmp/percona")
				t.Setenv(telemetryCheckInterval, strconv.Itoa(telemetryCheckIntervalDefault*2))
				t.Setenv(telemetryResendInterval, strconv.Itoa(telemetryResendIntervalDefault*3))
				t.Setenv(telemetryHistoryKeepInterval, strconv.Itoa(historyKeepIntervalDefault*4))
				t.Setenv(telemetryURL, "https://check.percona.com/v1/telemetry/GenericReport2")
			},
			expectedConfig: Config{
				PSMetricsPath:                filepath.Join("/tmp", "percona", "ps"),
				PSMDBMetricsPath:             filepath.Join("/tmp", "percona", "psmdb"),
				PXCMetricsPath:               filepath.Join("/tmp", "percona", "pxc"),
				PGMetricsPath:                filepath.Join("/tmp", "percona", "pg"),
				TelemetryCheckInterval:       telemetryCheckIntervalDefault * 2,
				TelemetryResendTimeout:       telemetryResendIntervalDefault * 3,
				TelemetryHistoryPath:         filepath.Join("/tmp", "percona", "history"),
				TelemetryHistoryKeepInterval: historyKeepIntervalDefault * 4,
				PerconaTelemetryURL:          "https://check.percona.com/v1/telemetry/GenericReport2",
			},
		},
		{
			name: "redefine_partial_values",
			setupTestData: func(t *testing.T) {
				t.Helper()

				t.Setenv(telemetryCheckInterval, strconv.Itoa(telemetryCheckIntervalDefault*2))
				t.Setenv(telemetryResendInterval, strconv.Itoa(telemetryResendIntervalDefault*3))
				t.Setenv(telemetryURL, "https://check-dev.percona.com/v1/telemetry/GenericReport2")
			},
			expectedConfig: Config{
				PSMetricsPath:                filepath.Join("/usr", "local", "percona", "telemetry", "ps"),
				PSMDBMetricsPath:             filepath.Join("/usr", "local", "percona", "telemetry", "psmdb"),
				PXCMetricsPath:               filepath.Join("/usr", "local", "percona", "telemetry", "pxc"),
				PGMetricsPath:                filepath.Join("/usr", "local", "percona", "telemetry", "pg"),
				TelemetryCheckInterval:       telemetryCheckIntervalDefault * 2,
				TelemetryResendTimeout:       telemetryResendIntervalDefault * 3,
				TelemetryHistoryPath:         filepath.Join("/usr", "local", "percona", "telemetry", "history"),
				TelemetryHistoryKeepInterval: historyKeepIntervalDefault,
				PerconaTelemetryURL:          "https://check-dev.percona.com/v1/telemetry/GenericReport2",
			},
		},
	}

	for _, tt := range testCases { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			// resetting viper environment variables
			t.Cleanup(func() {
				viper.Reset()
			})

			tt.setupTestData(t)
			gotConfig := InitConfig()
			require.Equal(t, tt.expectedConfig, gotConfig)
		})
	}
}
