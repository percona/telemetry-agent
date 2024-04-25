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
	"os"
	"path/filepath"
	"strings"
	"testing"

	platformReporter "github.com/percona-platform/saas/gen/telemetry/generic"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func writeTempFiles(t *testing.T, path string, files ...string) {
	t.Helper()
	for _, file := range files {
		err := os.WriteFile(filepath.Join(path, file), []byte(file), metricsFilePermissions)
		require.NoError(t, err)
	}
}

func checkFilesExist(t *testing.T, path string, files ...string) {
	t.Helper()
	for _, file := range files {
		filePath := filepath.Join(path, file)
		_, err := os.Stat(filePath)
		require.NoError(t, err)
	}
}

func checkFilesAbsent(t *testing.T, path string, files ...string) {
	t.Helper()
	for _, file := range files {
		filePath := filepath.Join(path, file)
		_, err := os.Stat(filePath)
		require.Error(t, err)
	}
}

func checkInstanceIDInFile(t *testing.T, path, fileName, wantInstanceID string) {
	t.Helper()
	filePath := filepath.Join(path, fileName)
	file, err := os.Open(filepath.Clean(filePath))
	require.NoError(t, err)
	// do not forget to close file.
	defer file.Close() //nolint:errcheck

	var instanceID string
	scanner := bufio.NewScanner(file)
	scanner.Split(customSplitFunc)
	for scanner.Scan() {
		if parts := strings.Split(scanner.Text(), ":"); len(parts) == 2 && parts[0] == InstanceIDKey {
			instanceID = strings.TrimSpace(parts[1])
			break
		}
	}

	require.NoError(t, scanner.Err())
	require.Equal(t, wantInstanceID, instanceID)
}

func checkDirectoryContentCount(t *testing.T, tmpDir string, wantCount int) {
	t.Helper()
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, files, wantCount)
}

func checkHistoryFileContent(t *testing.T, tmpDir, historyFile string, req *platformReporter.ReportRequest) {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Clean(filepath.Join(tmpDir, historyFile)))
	require.NoError(t, err)
	require.NotEmpty(t, bytes)

	result := &platformReporter.ReportRequest{}
	err = protojson.Unmarshal(bytes, result)
	require.NoError(t, err)
	require.Equal(t, result, req)
}
