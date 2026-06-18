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
	// Percona env variable that contains OS name in docker container.
	dockerOSEnv = "OS_VER"
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

func getInstanceID(instanceFile string) string {
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
	var err error

	dirName := filepath.Dir(cleanInstanceFile)

	_, err = os.Stat(dirName)
	if os.IsNotExist(err) {
		// directory is absent, creating
		err = os.MkdirAll(dirName, os.ModePerm|0o775)
		if err != nil {
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

	file, err = os.Open(cleanInstanceFile)
	if err != nil {
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
	defer func(l *zap.SugaredLogger) {
		fErr := file.Close()
		if fErr != nil {
			l.Errorw("failed to close file", zap.Error(fErr))
		}
	}(l)

	st, err := file.Stat()
	if err != nil || st.Size() == 0 {
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

	err = scanner.Err()
	if err != nil {
		l.Warnw("failed to read instanceId from Percona telemetry file, fallback to random UUID", zap.Error(err))
		// fallback to random UUID
		createTelemetryFile(cleanInstanceFile, newInstanceID)

		return newInstanceID
	}

	err = uuid.Validate(instanceID)
	if err != nil {
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
	err := os.WriteFile(instanceFile, fmt.Appendf(nil, "%s:%s\n", InstanceIDKey, instanceID), metricsFilePermissions)
	if err != nil {
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
	if getDeploymentInfo() == deploymentDocker {
		if val, found := os.LookupEnv(dockerOSEnv); found {
			return val
		}
	}

	filePath := filepath.Join("/etc", "os-release")

	_, err := os.Stat(filePath)
	if err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readOSReleaseFile(filePath)
	}

	filePath = filepath.Join("/etc", "system-release")

	_, err = os.Stat(filePath)
	if err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readSystemReleaseFile(filePath)
	}

	filePath = filepath.Join("/etc", "redhat-release")

	_, err = os.Stat(filePath)
	if err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readSystemReleaseFile(filePath)
	}

	filePath = filepath.Join("/etc", "issue")

	_, err = os.Stat(filePath)
	if err == nil {
		zap.L().Sugar().Debugw("getting OS info from file", zap.String("file", filePath))
		return readSystemReleaseFile(filePath)
	}

	return unknownString
}

func getHardwareInfo(ctx context.Context) string {
	var (
		unamePath string
		err       error
	)

	unamePath, err = exec.LookPath("uname")
	if err != nil {
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
	l := zap.L().Sugar().With(zap.String("file", cleanFileName))

	f, err := os.Open(cleanFileName)
	if err != nil {
		l.Errorw("failed to open OS file", zap.Error(err), zap.String("file", cleanFileName))
		return unknownString
	}

	defer func(l *zap.SugaredLogger) {
		fErr := f.Close()
		if fErr != nil {
			l.Errorw("failed to close file", zap.Error(fErr))
		}
	}(l)

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

	err = scanner.Err()
	if err != nil {
		l.Warnw("error reading OS release file", zap.String("file", cleanFileName), zap.Error(err))
		return unknownString
	}

	return unknownString
}

func readSystemReleaseFile(fileName string) string {
	cleanFileName := filepath.Clean(fileName)
	l := zap.L().Sugar().With(zap.String("file", cleanFileName))

	f, err := os.Open(cleanFileName)
	if err != nil {
		l.Errorw("failed to open system release file", zap.Error(err), zap.String("file", cleanFileName))
		return unknownString
	}

	defer func(l *zap.SugaredLogger) {
		fErr := f.Close()
		if fErr != nil {
			l.Errorw("failed to close file", zap.Error(fErr))
		}
	}(l)

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.Trim(scanner.Text(), `"`)
	}

	err = scanner.Err()
	if err != nil {
		l.Warnw("error reading system release file", zap.String("file", cleanFileName), zap.Error(err))
		return unknownString
	}

	return unknownString
}
