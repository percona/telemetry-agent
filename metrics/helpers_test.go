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
	"testing"

	platformReporter "github.com/percona-platform/platform/gen/telemetry/generic"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func writeTempFiles(t *testing.T, path string, files ...string) {
	t.Helper()
	for _, file := range files {
		err := os.WriteFile(filepath.Join(path, file), []byte(file), 0o600)
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

func checkDirectoryContentCount(t *testing.T, tmpDir string, wantCount int) {
	t.Helper()
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Equal(t, len(files), wantCount)
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
