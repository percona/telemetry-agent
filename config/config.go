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
	perconaTelemetryEnvPrefix      = "PERCONA_TELEMETRY"
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
	viper.SetEnvPrefix(perconaTelemetryEnvPrefix)
	var err error

	pillarMetricsRootPathVarName := "root_path"

	pillarMetricsRootPathDefault := filepath.Join("/usr", "local", "percona", "telemetry")
	err = viper.BindEnv(pillarMetricsRootPathVarName)
	if err != nil {
		panic(err)
	}
	viper.SetDefault(pillarMetricsRootPathVarName, pillarMetricsRootPathDefault)
	telemetryRootPath := viper.GetString(pillarMetricsRootPathVarName)

	telemetryCheckIntervalVarName := "check_interval"
	err = viper.BindEnv(telemetryCheckIntervalVarName)
	if err != nil {
		panic(err)
	}
	viper.SetDefault(telemetryCheckIntervalVarName, telemetryCheckIntervalDefault)

	telemetryResendIntervalVarName := "resend_interval"
	err = viper.BindEnv(telemetryResendIntervalVarName)
	if err != nil {
		panic(err)
	}
	viper.SetDefault(telemetryResendIntervalVarName, telemetryResendIntervalDefault)

	historyKeepIntervalVarName := "history_keep_interval"
	err = viper.BindEnv(historyKeepIntervalVarName)
	if err != nil {
		panic(err)
	}
	viper.SetDefault(historyKeepIntervalVarName, historyKeepIntervalDefault)

	telemetryURLVarName := "url"
	err = viper.BindEnv(telemetryURLVarName)
	if err != nil {
		panic(err)
	}
	viper.SetDefault(telemetryURLVarName, perconaTelemetryURLDefault)

	return Config{
		PSMetricsPath:                filepath.Join(telemetryRootPath, "ps"),
		PSMDBMetricsPath:             filepath.Join(telemetryRootPath, "psmdb"),
		PXCMetricsPath:               filepath.Join(telemetryRootPath, "pxc"),
		PGMetricsPath:                filepath.Join(telemetryRootPath, "pg"),
		TelemetryCheckInterval:       viper.GetInt(telemetryCheckIntervalVarName),
		TelemetryHistoryPath:         filepath.Join(telemetryRootPath, "history"),
		TelemetryResendTimeout:       viper.GetInt(telemetryResendIntervalVarName),
		TelemetryHistoryKeepInterval: viper.GetInt(historyKeepIntervalVarName),
		PerconaTelemetryURL:          viper.GetString(telemetryURLVarName),
	}
}
