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
	"net/url"
	"path/filepath"

	"github.com/alecthomas/kong"
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

//nolint:gochecknoglobals
var (
	Version   string
	Commit    string
	BuildDate string
)

// TelemetryOpts represents the options for configuring telemetry paths on local filesystem.
type TelemetryOpts struct {
	RootPath      string `help:"define Percona telemetry root path on local filesystem." env:"PERCONA_TELEMETRY_ROOT_PATH" default:"/usr/local/percona/telemetry"`
	PSMetricsPath string `kong:"-"`
	// For PSMDB (mongod) component
	PSMDBMetricsPath string `kong:"-"`
	// For PSMDB (mongos) component
	PSMDBSMetricsPath   string `kong:"-"`
	PXCMetricsPath      string `kong:"-"`
	PGMetricsPath       string `kong:"-"`
	HistoryPath         string `kong:"-"`
	CheckInterval       int    `help:"define time interval in seconds for checking Percona Pillars telemetry." env:"PERCONA_TELEMETRY_CHECK_INTERVAL" default:"86400"`
	HistoryKeepInterval int    `help:"define time interval in seconds for keeping old history telemetry files on filesystem." env:"PERCONA_TELEMETRY_HISTORY_KEEP_INTERVAL" default:"604800"`
}

// PlatformOpts represents the options for configuring communication with Percona Platform parameters.
type PlatformOpts struct {
	ResendTimeout int    `help:"define wait time in seconds to sleep before retrying request to Percona Platform in case of request failure." env:"PERCONA_TELEMETRY_RESEND_INTERVAL" default:"60"`
	URL           string `help:"define Percona Platform URL for sending Pillars telemetry to." env:"PERCONA_TELEMETRY_URL" default:"https://check.percona.com/v1/telemetry/GenericReport"`
}

// LogOpts represents the options for configuring logging.
type LogOpts struct {
	Verbose bool `help:"enable verbose logging." default:"false"`
	DevMode bool `help:"enable development mode logging." default:"false"`
}

// Config struct used for storing Telemetry Agent configuration parameters.
type Config struct {
	Telemetry TelemetryOpts `embed:"" prefix:"telemetry."`
	Platform  PlatformOpts  `embed:"" prefix:"platform."`
	Log       LogOpts       `embed:"" prefix:"log."`
	Version   bool          `help:"Show version and exit"`
}

// InitConfig parses Telemetry Agent configuration parameters.
// If some parameters are not defined - default values are used instead.
func InitConfig() Config {
	var conf Config
	ctx := kong.Parse(&conf,
		kong.Name("telemetry-agent"),
		kong.Description("Percona Telemetry Agent gathers information from running Percona Pillar products, about the host and installed Percona software and sends it to Percona Platform."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": Version,
		},
	)

	if len(conf.Telemetry.RootPath) == 0 {
		ctx.Fatalf("No telemetry root path was specified. You must specify the path with the --telemetry.rootPath command argument or the PERCONA_TELEMETRY_ROOT_PATH environment variable")
	}

	// Validate URL
	if len(conf.Platform.URL) == 0 {
		ctx.Fatalf("No Percona Platform URL was specified for sending Pillars telemetry. You must specify the path with the --platform.url command argument or the PERCONA_TELEMETRY_URL environment variable")
	}

	u, err := url.ParseRequestURI(conf.Platform.URL)
	if err != nil {
		ctx.Fatalf("Invalid Percona Platform Telemetry URL: %q", err)
	}
	if u.Scheme == "" || u.Host == "" {
		ctx.Fatalf("Invalid Percona Platform Telemetry URL: scheme or host is missed")
	}

	conf.Telemetry.PSMetricsPath = filepath.Join(conf.Telemetry.RootPath, "ps")
	conf.Telemetry.PSMDBMetricsPath = filepath.Join(conf.Telemetry.RootPath, "psmdb")
	conf.Telemetry.PSMDBSMetricsPath = filepath.Join(conf.Telemetry.RootPath, "psmdbs")
	conf.Telemetry.PXCMetricsPath = filepath.Join(conf.Telemetry.RootPath, "pxc")
	conf.Telemetry.PGMetricsPath = filepath.Join(conf.Telemetry.RootPath, "pg")
	conf.Telemetry.HistoryPath = filepath.Join(conf.Telemetry.RootPath, "history")
	return conf
}
