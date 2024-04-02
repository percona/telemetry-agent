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

// Package logger provides functions for configuring global logger.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GlobalOpts contains logger options.
type GlobalOpts struct {
	LogDebug   bool   // enable debug level logging
	LogDevMode bool   // enable development mode logging: text instead of JSON, DPanic panics instead of logging errors
	LogName    string // global logger name
}

// SetupGlobal setups global zap logger.
func SetupGlobal(opts *GlobalOpts) {
	// catch the common service initialization problem
	if opts == nil {
		opts = &GlobalOpts{}
	}

	cfg := &zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if opts.LogDebug {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	if opts.LogDevMode {
		cfg.Development = true
		cfg.Encoding = "console"
		cfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(l.Named(opts.LogName))
}
