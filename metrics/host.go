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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/matishsiao/goInfo"
	"go.uber.org/zap"
)

const (

	// InstanceIDKey key name in telemetryFile with host instance ID.
	InstanceIDKey     = "instanceId"
	unknownOS         = "unknown"
	telemetryFile     = "/usr/local/percona/telemetry_uuid"
	deploymentPackage = "PACKAGE"
	deploymentDocker  = "DOCKER"
	perconaDockerEnv  = "FULL_PERCONA_VERSION"
)

// ScrapeHostMetrics gathers metrics about host where Telemetry Agent is running.
// In addition, it checks Percona telemetry file and extracts instanceId value from it.
func ScrapeHostMetrics() (*File, error) {
	instanceID, err := getInstanceID(telemetryFile)
	if err != nil {
		return nil, fmt.Errorf("can't get Percona telemetry instanceID: %w", err)
	}
	m := &File{
		Timestamp: time.Now(),
		Filename:  telemetryFile,
	}
	m.Metrics = make(map[string]string)
	m.Metrics[InstanceIDKey] = instanceID

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

func getInstanceID(instanceFile string) (string, error) { //nolint:cyclop
	l := zap.L().Sugar().With(zap.String("file", instanceFile))
	l.Debug("processing Percona telemetry file")

	// Notes: Percona telemetry file (/usr/local/percona/telemetry_uuid) or directory
	// may be absent. In such a case this file shall be created with the following content:
	// "instanceId: <uuid>"
	// example:
	// "instanceId: 1bed5f0d-cc3a-11ee-bd8a-c84bd64e0277".
	cleanInstanceFile := filepath.Clean(instanceFile)
	dirName := filepath.Dir(cleanInstanceFile)
	_, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		// directory is absent, creating
		if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
			l.Errorw("can't create directory",
				zap.String("directory", dirName),
				zap.Error(err))
			return "", err
		}
		return createTelemetryFile(cleanInstanceFile)
	}

	var instanceID string

	file, err := os.Open(cleanInstanceFile)
	// do not forget to close file.
	defer func() {
		_ = file.Close()
	}()

	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			l.Errorw("failed to read Percona telemetry file, skipping", zap.Error(err))
			return "", err
		}
		// telemetry file is absent, fill values on our own
		// and write back to file.
		return createTelemetryFile(cleanInstanceFile)
	} else if st, err := file.Stat(); err != nil {
		l.Errorw("failed to get file info", zap.Error(err))
		return "", err
	} else if st.Size() == 0 {
		// file exists but is empty
		return createTelemetryFile(cleanInstanceFile)
	}

	// file exists and is not empty.
	// get "instanceID" value from file.
	scanner := bufio.NewScanner(file)
	scanner.Split(customSplitFunc)
	for scanner.Scan() {
		if parts := strings.Split(scanner.Text(), ":"); len(parts) == 2 && parts[0] == InstanceIDKey {
			instanceID = strings.TrimSpace(parts[1])
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading Percona telemetry file, scanner error: %w", err)
	}

	if len(instanceID) == 0 {
		l.Error("failed to get Percona telemetry instanceID, it is empty")
	}
	return instanceID, nil
}

func createTelemetryFile(instanceFile string) (string, error) {
	instanceID := uuid.New().String()
	if err := os.WriteFile(instanceFile, []byte(fmt.Sprintf("%s: %s", InstanceIDKey, instanceID)), 0o600); err != nil {
		zap.L().Sugar().With(zap.String("file", instanceFile)).
			Errorw("failed to write Percona telemetry file", zap.Error(err))
		return "", err
	}
	return instanceID, nil
}

func getDeploymentInfo() string {
	if _, found := os.LookupEnv(perconaDockerEnv); found {
		return deploymentDocker
	}
	return deploymentPackage
}

func getOSInfo() string {
	filePath := filepath.Join("/etc", "os-release")
	if _, err := os.Stat(filePath); err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readOSReleaseFile(filePath)
	}

	filePath = filepath.Join("/etc", "system-release")
	if _, err := os.Stat(filePath); err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readSystemReleaseFile(filePath)
	}

	filePath = filepath.Join("/etc", "redhat-release")
	if _, err := os.Stat(filePath); err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readSystemReleaseFile(filePath)
	}

	filePath = filepath.Join("/etc", "issue")
	if _, err := os.Stat(filePath); err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readSystemReleaseFile(filePath)
	}
	return unknownOS
}

func getHardwareInfo() string {
	gi, err := goInfo.GetInfo()
	if err != nil {
		return "unknown"
	}
	return gi.Platform
}

func readOSReleaseFile(fileName string) string {
	f, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return unknownOS
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				return strings.Trim(parts[1], `"`)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("error reading os release file", zap.String("file", fileName), zap.Error(err))
		return unknownOS
	}

	return unknownOS
}

func readSystemReleaseFile(fileName string) string {
	f, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return unknownOS
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.Trim(scanner.Text(), `"`)
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("error reading system release file", zap.String("file", fileName), zap.Error(err))
		return unknownOS
	}
	return unknownOS
}
