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

package metrics

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	platformReporter "github.com/percona-platform/platform/gen/telemetry/generic"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// WriteMetricsToHistory creates a new telemetry history file and writes the content of
// Percona Platform telemetry request into it. Content is written using JSON format.
func WriteMetricsToHistory(historyFile string, platformReport *platformReporter.ReportRequest) error {
	l := zap.L().Sugar()
	if platformReport == nil || len(platformReport.Reports) == 0 {
		l.Errorw("attempt to write invalid Percona Platform report into history file",
			zap.Any("report", platformReport))
		return errors.New("invalid telemetry report, ReportRequest.Reports is empty")
	}

	cleanFilePath := filepath.Clean(historyFile)
	// check that directory exists
	dirPath := filepath.Dir(cleanFilePath)
	if err := validateDirectory(dirPath); err != nil {
		return errors.Wrap(err, "can't read directory with history metric files")
	}

	// Marshal the message to pretty JSON
	marshalOpts := protojson.MarshalOptions{Indent: "  ", UseProtoNames: false}
	jsonBytes, err := marshalOpts.Marshal(platformReport)
	if err != nil {
		l.Errorw("failed to marshal Percona Platform report into JSON", zap.Error(err))
		return errors.Wrap(err, "can't marshal report into JSON")
	}

	if err := os.WriteFile(cleanFilePath, jsonBytes, 0o600); err != nil {
		l.Errorw("failed to write history file",
			zap.String("file", historyFile),
			zap.Error(err))
		return errors.Wrap(err, "can't write history file")
	}
	return nil
}

// CleanupMetricsHistory removes all telemetry files from history directory that are older than threshold.
// File creation time is taken from file name - it contains unixtime in format:
// <unixtime>-<random token>.json.
func CleanupMetricsHistory(historyDirectoryPath string, keepInterval int) error {
	l := zap.L().Sugar()

	cleanHistoryPath := filepath.Clean(historyDirectoryPath)
	// check that directory exists
	if err := validateDirectory(cleanHistoryPath); err != nil {
		return errors.Wrap(err, "can't read directory with history metric files")
	}

	files, err := os.ReadDir(cleanHistoryPath)
	if err != nil {
		return errors.Wrap(err, "can't read directory with history metric files")
	}

	timeThreshold := time.Now().Add(-time.Duration(keepInterval) * time.Second)
	for _, file := range files {
		fl := l.With(zap.String("file", filepath.Join(cleanHistoryPath, file.Name())))

		fileExt := filepath.Ext(file.Name())
		if !file.Type().IsRegular() || fileExt != ".json" {
			fl.Debug("seems not a metrics file, skipping")
			continue
		}

		fileCreationTime, err := strconv.Atoi(strings.Split(
			strings.TrimSuffix(filepath.Base(file.Name()), fileExt),
			"-")[0])
		if err != nil {
			fl.Warnw("can't convert filename into int, skipping", zap.Error(err))
			continue
		}

		t := time.Unix(int64(fileCreationTime), 0)
		if t.After(timeThreshold) {
			fl.Debugw("file age threshold is not reached, skipping",
				zap.Time("creationTime", t),
				zap.Time("threshold", timeThreshold))
			continue
		}

		fl.Debug("removing file")
		if err := os.Remove(filepath.Join(cleanHistoryPath, file.Name())); err != nil {
			fl.Errorw("error removing metric file, skipping", zap.Error(err))
			continue
		}
	}
	return nil
}

func validateDirectory(dirPath string) error {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return err
	}
	if !info.IsDir() {
		return errors.New("provided path is not a directory")
	}
	return nil
}
