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

// Package config provides functionality for processing Telemetry Agent configuration parameters.
package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	telemetryRootPath              = "PERCONA_TELEMETRY_ROOT_PATH"
	telemetryCheckInterval         = "PERCONA_TELEMETRY_CHECK_INTERVAL"
	telemetryResendInterval        = "PERCONA_TELEMETRY_RESEND_INTERVAL"
	telemetryHistoryKeepInterval   = "PERCONA_TELEMETRY_HISTORY_KEEP_INTERVAL"
	telemetryURL                   = "PERCONA_TELEMETRY_URL"
	telemetryCheckIntervalDefault  = 24 * 60 * 60     // seconds
	telemetryResendIntervalDefault = 60               // seconds
	historyKeepIntervalDefault     = 7 * 24 * 60 * 60 // 7d
	perconaTelemetryURLDefault     = "https://check.percona.com/v1/telemetry/GenericReport"
)

// Config struct used for storing Telemetry Agent configuration parameters.
type Config struct {
	PSMetricsPath                string
	PSMDBMetricsPath             string
	PXCMetricsPath               string
	PGMetricsPath                string
	TelemetryCheckInterval       int
	TelemetryResendTimeout       int
	TelemetryHistoryPath         string
	TelemetryHistoryKeepInterval int
	PerconaTelemetryURL          string
}

// InitConfig parses Telemetry Agent configuration parameters.
// If some parameters are not defined - default values are used instead.
func InitConfig() Config {
	// viper.SetEnvPrefix(perconaTelemetryEnvPrefix)

	viper.MustBindEnv(telemetryRootPath)
	viper.SetDefault(telemetryRootPath, filepath.Join("/usr", "local", "percona", "telemetry"))
	rootPathValue := viper.GetString(telemetryRootPath)

	viper.MustBindEnv(telemetryCheckInterval)
	viper.SetDefault(telemetryCheckInterval, telemetryCheckIntervalDefault)

	viper.MustBindEnv(telemetryResendInterval)
	viper.SetDefault(telemetryResendInterval, telemetryResendIntervalDefault)

	viper.MustBindEnv(telemetryHistoryKeepInterval)
	viper.SetDefault(telemetryHistoryKeepInterval, historyKeepIntervalDefault)

	viper.MustBindEnv(telemetryURL)
	viper.SetDefault(telemetryURL, perconaTelemetryURLDefault)

	return Config{
		PSMetricsPath:                filepath.Join(rootPathValue, "ps"),
		PSMDBMetricsPath:             filepath.Join(rootPathValue, "psmdb"),
		PXCMetricsPath:               filepath.Join(rootPathValue, "pxc"),
		PGMetricsPath:                filepath.Join(rootPathValue, "pg"),
		TelemetryCheckInterval:       viper.GetInt(telemetryCheckInterval),
		TelemetryHistoryPath:         filepath.Join(rootPathValue, "history"),
		TelemetryResendTimeout:       viper.GetInt(telemetryResendInterval),
		TelemetryHistoryKeepInterval: viper.GetInt(telemetryHistoryKeepInterval),
		PerconaTelemetryURL:          viper.GetString(telemetryURL),
	}
}
