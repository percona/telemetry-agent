/*
 * // Copyright (C) 2024 Percona LLC
 * //
 * // This program is free software: you can redistribute it and/or modify
 * // it under the terms of the GNU Affero General Public License as published by
 * // the Free Software Foundation, either version 3 of the License, or
 * // (at your option) any later version.
 * //
 * // This program is distributed in the hope that it will be useful,
 * // but WITHOUT ANY WARRANTY; without even the implied warranty of
 * // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * // GNU Affero General Public License for more details.
 * //
 * // You should have received a copy of the GNU Affero General Public License
 * // along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/matishsiao/goInfo"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	telemetryFile = "/usr/local/percona/telemetry_uuid"
	// key name in telemetryFile with host instance ID.
	instanceIDKey = "instanceId"
)

// ScrapeHostMetrics gathers metrics about host where Telemetry Agent is running.
// In addition, it checks Percona telemetry file and extracts instanceId value from it.
func ScrapeHostMetrics() (*File, error) {
	instanceID, err := getInstanceID()
	if err != nil {
		return nil, errors.Wrap(err, "can't get Percona telemetry instanceID")
	}
	m := &File{
		Timestamp: time.Now(),
		Filename:  telemetryFile,
	}
	m.Metrics = make(map[string]string)
	m.Metrics[instanceIDKey] = instanceID

	m.Metrics["OS"] = getOSInfo()
	m.Metrics["deployment"] = getDeploymentInfo()
	m.Metrics["hardware_arch"] = getHardwareInfo()

	return m, nil
}

func customSplitFunc(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if atEOF {
		return len(data), data, nil
	}

	if i := strings.Index(string(data), "\n"); i >= 0 {
		// skip the delimiter in advancing to the next pair
		return i + 1, data[0:i], nil
	}
	return 0, nil, nil
}

func getInstanceID() (string, error) {
	l := zap.L().Sugar().With(zap.String("file", telemetryFile))
	l.Debug("processing Percona telemetry file")

	// Notes: Percona telemetry file (/usr/local/percona/telemetry_uuid)
	// may be absent. In such a case this file shall be created with the following content:
	// "instanceId: <uuid>"
	// example:
	// "instanceId: 1bed5f0d-cc3a-11ee-bd8a-c84bd64e0277".
	var instanceID string
	if file, err := os.Open(telemetryFile); err != nil { //nolint:nestif
		if !errors.Is(err, os.ErrNotExist) {
			l.Errorw("failed to read Percona telemetry file, skipping", zap.Error(err))
			return "", err
		}
		// telemetry file is absent, fill values on our own
		// and write back to file
		instanceID = uuid.New().String()

		l.Info("Percona telemetry file is absent, creating new one")

		if err := os.WriteFile(telemetryFile, []byte(fmt.Sprintf("%s: %s", instanceIDKey, instanceID)), 0o600); err != nil {
			l.Errorw("failed to write Percona telemetry file", zap.Error(err))
			return "", err
		}
	} else {
		// do not forget to close file
		defer func(file *os.File, fl *zap.SugaredLogger) {
			err := file.Close()
			if err != nil {
				fl.Errorw("failed to close Percona telemetry file", zap.Error(err))
			}
		}(file, l)

		// get "instanceID" value from file
		scanner := bufio.NewScanner(file)
		scanner.Split(customSplitFunc)
		for scanner.Scan() {
			if parts := strings.Split(scanner.Text(), ":"); len(parts) == 2 && parts[0] == instanceIDKey {
				instanceID = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if len(instanceID) == 0 {
		l.Error("failed to get Percona telemetry instanceID, it is empty")
	}
	return instanceID, nil
}

func getDeploymentInfo() string {
	// TODO: determine environment
	return "PACKAGE"
}

func getOSInfo() string {
	gi, err := goInfo.GetInfo()
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%s %s", gi.OS, gi.Core)
}

func getHardwareInfo() string {
	gi, err := goInfo.GetInfo()
	if err != nil {
		return "unknown"
	}
	return gi.Platform
}
