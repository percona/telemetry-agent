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

// Package metrics provides functionality for processing Percona Pillar's metrics.
package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	platformReporter "github.com/percona-platform/platform/gen/telemetry/generic"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// File struct used for storing parsed Pillar's or host metrics.
// One object hold info about of metrics file.
type File struct {
	Filename      string
	Timestamp     time.Time
	ProductFamily platformReporter.ProductFamily
	Metrics       map[string]string
}

func processMetricsDirectory(path string, productFamily platformReporter.ProductFamily) ([]*File, error) {
	l := zap.L().Sugar()

	toReturn := make([]*File, 0, 5)

	cleanMetricsDirectoryPath := filepath.Clean(path)
	files, err := os.ReadDir(cleanMetricsDirectoryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			l.Infow("pillar metric directory is absent, skipping", zap.String("directory", cleanMetricsDirectoryPath))
			return toReturn, nil
		}
		l.Errorw("failed to read pillar metric directory",
			zap.String("directory", cleanMetricsDirectoryPath),
			zap.Error(err))
		return nil, errors.Wrap(err, "can't read directory with metric files")
	}

	if len(files) == 0 {
		l.Infow("pillar metric directory is empty, skipping", zap.String("directory", cleanMetricsDirectoryPath))
		return toReturn, nil
	}

	for _, file := range files {
		fileName := filepath.Join(cleanMetricsDirectoryPath, file.Name())
		fl := l.With(zap.String("file", fileName))

		fileExt := filepath.Ext(file.Name())
		if !file.Type().IsRegular() || fileExt != ".json" {
			fl.Debug("seems not a metrics file, skipping")
			continue
		}

		fl.Debugw("parsing metrics file")
		fileMetrics, err := parseMetricsFile(fileName)
		if err != nil {
			fl.Errorw("error during parsing metrics file, skipping", zap.Error(err))
			continue
		}
		fileMetrics.ProductFamily = productFamily
		toReturn = append(toReturn, fileMetrics)
	}

	return toReturn, nil
}

func parseMetricsFile(path string) (*File, error) {
	cleanPath := filepath.Clean(path)
	fl := zap.L().Sugar().With(zap.String("file", cleanPath))

	file, err := os.Open(cleanPath)
	if err != nil {
		fl.Errorw("error during opening metrics file", zap.Error(err))
		return nil, err
	}
	defer func(file *os.File, fl *zap.SugaredLogger) {
		err := file.Close()
		if err != nil {
			fl.Errorw("error during closing metrics file", zap.Error(err))
		}
	}(file, fl)

	// file has content in JSON format but the structure is not well known beforehand.
	var tmpMetrics map[string]interface{}
	err = json.NewDecoder(file).Decode(&tmpMetrics)
	if err != nil {
		fl.Errorw("error during parsing metrics file, skipping", zap.Error(err))
		return nil, err
	}

	metrics := make(map[string]string)

	for k, v := range tmpMetrics {
		s, err := json.Marshal(v)
		if err != nil {
			fl.Errorw("error during marshalling metric value to JSON, skipping",
				zap.Any("value", v),
				zap.Error(err))
			return nil, err
		}
		metrics[k] = string(s)
	}
	// get timestamp from filename.
	// filename has format: <timestamp>-<random token>.json
	// example: 1708026156-d7664a58-d855-45c9-b017-50678cf620bb.json
	fileCreationTime, err := strconv.Atoi(strings.Split(
		strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name())),
		"-")[0])
	if err != nil {
		fl.Errorw("can't convert filename into int, skipping", zap.Error(err))
		return nil, err
	}

	return &File{
		Filename:  path,
		Timestamp: time.Unix(int64(fileCreationTime), 0),
		Metrics:   metrics,
	}, nil
}
