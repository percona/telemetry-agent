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
	"os"
	"path/filepath"
	"strconv"
	"testing"

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
				os.Args = []string{""}
			},
			expectedConfig: Config{
				Telemetry: TelemetryOpts{
					RootPath:            filepath.Join("/usr", "local", "percona", "telemetry"),
					PSMetricsPath:       filepath.Join("/usr", "local", "percona", "telemetry", "ps"),
					PSMDBMetricsPath:    filepath.Join("/usr", "local", "percona", "telemetry", "psmdb"),
					PSMDBSMetricsPath:   filepath.Join("/usr", "local", "percona", "telemetry", "psmdbs"),
					PXCMetricsPath:      filepath.Join("/usr", "local", "percona", "telemetry", "pxc"),
					PGMetricsPath:       filepath.Join("/usr", "local", "percona", "telemetry", "pg"),
					CheckInterval:       telemetryCheckIntervalDefault,
					HistoryPath:         filepath.Join("/usr", "local", "percona", "telemetry", "history"),
					HistoryKeepInterval: historyKeepIntervalDefault,
				},
				Platform: PlatformOpts{
					ResendTimeout: telemetryResendIntervalDefault,
					URL:           perconaTelemetryURLDefault,
				},
				Log: LogOpts{
					Verbose: false,
					DevMode: false,
				},
			},
		},
		{
			name: "redefine_all_values",
			setupTestData: func(t *testing.T) {
				t.Helper()

				os.Args = []string{""}
				t.Setenv(telemetryRootPath, "/tmp/percona")
				t.Setenv(telemetryCheckInterval, strconv.Itoa(telemetryCheckIntervalDefault*2))
				t.Setenv(telemetryResendInterval, strconv.Itoa(telemetryResendIntervalDefault*3))
				t.Setenv(telemetryHistoryKeepInterval, strconv.Itoa(historyKeepIntervalDefault*4))
				t.Setenv(telemetryURL, "https://check.percona.com/v1/telemetry/GenericReport2")
			},
			expectedConfig: Config{
				Telemetry: TelemetryOpts{
					RootPath:            filepath.Join("/tmp", "percona"),
					PSMetricsPath:       filepath.Join("/tmp", "percona", "ps"),
					PSMDBMetricsPath:    filepath.Join("/tmp", "percona", "psmdb"),
					PSMDBSMetricsPath:   filepath.Join("/tmp", "percona", "psmdbs"),
					PXCMetricsPath:      filepath.Join("/tmp", "percona", "pxc"),
					PGMetricsPath:       filepath.Join("/tmp", "percona", "pg"),
					CheckInterval:       telemetryCheckIntervalDefault * 2,
					HistoryPath:         filepath.Join("/tmp", "percona", "history"),
					HistoryKeepInterval: historyKeepIntervalDefault * 4,
				},
				Platform: PlatformOpts{
					ResendTimeout: telemetryResendIntervalDefault * 3,
					URL:           "https://check.percona.com/v1/telemetry/GenericReport2",
				},
				Log: LogOpts{
					Verbose: false,
					DevMode: false,
				},
			},
		},
		{
			name: "redefine_partial_values",
			setupTestData: func(t *testing.T) {
				t.Helper()

				os.Args = []string{""}
				t.Setenv(telemetryCheckInterval, strconv.Itoa(telemetryCheckIntervalDefault*2))
				t.Setenv(telemetryResendInterval, strconv.Itoa(telemetryResendIntervalDefault*3))
				t.Setenv(telemetryURL, "https://check-dev.percona.com/v1/telemetry/GenericReport2")
			},
			expectedConfig: Config{
				Telemetry: TelemetryOpts{
					RootPath:            filepath.Join("/usr", "local", "percona", "telemetry"),
					PSMetricsPath:       filepath.Join("/usr", "local", "percona", "telemetry", "ps"),
					PSMDBMetricsPath:    filepath.Join("/usr", "local", "percona", "telemetry", "psmdb"),
					PSMDBSMetricsPath:   filepath.Join("/usr", "local", "percona", "telemetry", "psmdbs"),
					PXCMetricsPath:      filepath.Join("/usr", "local", "percona", "telemetry", "pxc"),
					PGMetricsPath:       filepath.Join("/usr", "local", "percona", "telemetry", "pg"),
					CheckInterval:       telemetryCheckIntervalDefault * 2,
					HistoryPath:         filepath.Join("/usr", "local", "percona", "telemetry", "history"),
					HistoryKeepInterval: historyKeepIntervalDefault,
				},
				Platform: PlatformOpts{
					ResendTimeout: telemetryResendIntervalDefault * 3,
					URL:           "https://check-dev.percona.com/v1/telemetry/GenericReport2",
				},
				Log: LogOpts{
					Verbose: false,
					DevMode: false,
				},
			},
		},
	}

	for _, tt := range testCases { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			tt.setupTestData(t)
			gotConfig := InitConfig()
			require.Equal(t, tt.expectedConfig, gotConfig)
		})
	}
}
