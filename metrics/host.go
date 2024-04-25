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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (

	// InstanceIDKey key name in telemetryFile with host instance ID.
	InstanceIDKey     = "instanceId"
	unknownString     = "unknown"
	telemetryFile     = "/usr/local/percona/telemetry_uuid"
	deploymentPackage = "PACKAGE"
	deploymentDocker  = "DOCKER"
	perconaDockerEnv  = "FULL_PERCONA_VERSION"
)

// NOTE: the logic in this file is designed in a way "do our best to provide value", i.e. in case an error appears
// it is not passed to upper level but is just printed into log stream and fallback value is applied:
// - for instanceID it is random UUID
// - for OS it is "unknown"

// ScrapeHostMetrics gathers metrics about host where Telemetry Agent is running.
// In addition, it checks Percona telemetry file and extracts instanceId value from it.
func ScrapeHostMetrics(ctx context.Context) *File {
	f := &File{
		Timestamp: time.Now(),
		Filename:  telemetryFile,
	}
	f.Metrics = make(map[string]string)
	f.Metrics[InstanceIDKey] = getInstanceID(telemetryFile)
	f.Metrics["OS"] = getOSInfo()
	f.Metrics["deployment"] = getDeploymentInfo()
	f.Metrics["hardware_arch"] = getHardwareInfo(ctx)

	return f
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

func getInstanceID(instanceFile string) string { //nolint:cyclop
	cleanInstanceFile := filepath.Clean(instanceFile)
	l := zap.L().Sugar().With(zap.String("file", cleanInstanceFile))
	l.Debug("processing Percona telemetry file")

	newInstanceID := getRandomUUID()
	// Notes: Percona telemetry file (/usr/local/percona/telemetry_uuid) or directory
	// may be absent. In such a case this file shall be created with the following content:
	// "instanceId: <uuid>"
	// example:
	// "instanceId: 1bed5f0d-cc3a-11ee-bd8a-c84bd64e0277".
	//
	// In case of any error during file processing, new random instanceId is generated and
	// is written into telemetry file.
	dirName := filepath.Dir(cleanInstanceFile)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		// directory is absent, creating
		if err := os.MkdirAll(dirName, os.ModePerm|0o775); err != nil {
			l.Errorw("can't create directory, fallback to random UUID",
				zap.String("directory", dirName),
				zap.Error(err))
			// fallback to random UUID
			return newInstanceID
		}
		createTelemetryFile(cleanInstanceFile, newInstanceID)
		return newInstanceID
	}

	var file *os.File
	var err error
	if file, err = os.Open(cleanInstanceFile); err != nil {
		if !os.IsNotExist(err) {
			l.Errorw("failed to read Percona telemetry file, fallback to random UUID", zap.Error(err))
			// fallback to random UUID
			createTelemetryFile(cleanInstanceFile, newInstanceID)
			return newInstanceID
		}
		// telemetry file is absent, fill values on our own
		// and write back to file.
		l.Info("Percona telemetry file is absent, creating")
		createTelemetryFile(cleanInstanceFile, newInstanceID)
		return newInstanceID
	}

	// do not forget to close file.
	defer file.Close() //nolint:errcheck

	if st, err := file.Stat(); err != nil || st.Size() == 0 {
		l.Errorw("failed to get file info, fallback to random UUID", zap.Error(err))
		// fallback to random UUID
		createTelemetryFile(cleanInstanceFile, newInstanceID)
		return newInstanceID
	}

	// file exists and is not empty.
	// get "instanceID" value from file.
	var instanceID string
	scanner := bufio.NewScanner(file)
	scanner.Split(customSplitFunc)
	for scanner.Scan() {
		if parts := strings.Split(scanner.Text(), ":"); len(parts) == 2 && parts[0] == InstanceIDKey {
			instanceID = strings.TrimSpace(parts[1])
			break
		}
	}

	if err := scanner.Err(); err != nil {
		l.Warnw("failed to read instanceId from Percona telemetry file, fallback to random UUID", zap.Error(err))
		// fallback to random UUID
		createTelemetryFile(cleanInstanceFile, newInstanceID)
		return newInstanceID
	}

	if err := uuid.Validate(instanceID); err != nil {
		// "instanceID" is read from file, but it is invalid.
		l.Warn("failed to obtain Percona telemetry instanceID, fallback to random UUID")
		// fallback to random UUID
		createTelemetryFile(cleanInstanceFile, newInstanceID)
		return newInstanceID
	}
	return instanceID
}

func getRandomUUID() string {
	return uuid.New().String()
}

func createTelemetryFile(instanceFile, instanceID string) {
	if err := os.WriteFile(instanceFile, []byte(fmt.Sprintf("%s:%s\n", InstanceIDKey, instanceID)), metricsFilePermissions); err != nil {
		zap.L().Sugar().With(zap.String("file", instanceFile)).
			Errorw("failed to write Percona telemetry file", zap.Error(err))
	}
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
	return unknownString
}

func getHardwareInfo(ctx context.Context) string {
	var unamePath string
	var err error
	if unamePath, err = exec.LookPath("uname"); err != nil {
		zap.L().Sugar().Warnw("failed to get hardware info, uname binary is not found", zap.Error(err))
		return fmt.Sprintf("%s %s", unknownString, unknownString)
	}
	args := []string{unamePath, "-mp"}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseHardwareInfoOutput(outputB, err)
}

func parseHardwareInfoOutput(hwOutput []byte, hwErr error) string {
	if hwErr != nil {
		// If error is returned - something went wrong.
		zap.L().Sugar().Debugw("cmd output", zap.ByteString("output", hwOutput), zap.Error(hwErr))
		return fmt.Sprintf("%s %s", unknownString, unknownString)
	}
	scanner := bufio.NewScanner(bytes.NewReader(hwOutput))
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " '\t")
		if len(line) == 0 {
			continue
		}
		return line
	}
	return fmt.Sprintf("%s %s", unknownString, unknownString)
}

func readOSReleaseFile(fileName string) string {
	cleanFileName := filepath.Clean(fileName)
	f, err := os.Open(cleanFileName)
	if err != nil {
		zap.L().Sugar().Errorw("failed to open OS file", zap.Error(err), zap.String("file", cleanFileName))
		return unknownString
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
		zap.L().Sugar().Warnw("error reading OS release file", zap.String("file", cleanFileName), zap.Error(err))
		return unknownString
	}

	return unknownString
}

func readSystemReleaseFile(fileName string) string {
	cleanFileName := filepath.Clean(fileName)
	f, err := os.Open(cleanFileName)
	if err != nil {
		zap.L().Sugar().Errorw("failed to open system release file", zap.Error(err), zap.String("file", cleanFileName))
		return unknownString
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.Trim(scanner.Text(), `"`)
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("error reading system release file", zap.String("file", cleanFileName), zap.Error(err))
		return unknownString
	}
	return unknownString
}
