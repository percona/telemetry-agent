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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetInstanceID(t *testing.T) { //nolint:tparallel
	t.Parallel()

	tests := []struct {
		name              string
		setupTestData     func(t *testing.T, tmpDir, instanceFile, instanceID string) // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir, instanceFile string)             // Post CleanupMetricsHistory function validation
		wantErr           bool                                                        // Flags if we want the test to return an error
		newID             bool
	}{
		{
			name: "non_existing_directory",
			setupTestData: func(t *testing.T, tmpDir, instanceFile, instanceID string) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, instanceFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, instanceFile)

				data, err := os.ReadFile(filepath.Clean(filepath.Join(tmpDir, instanceFile)))
				require.NoError(t, err)
				require.Contains(t, string(data), "instanceId: ")
			},
			wantErr: false,
			newID:   true,
		},
		{
			name: "non_existing_file",
			setupTestData: func(t *testing.T, tmpDir, instanceFile, instanceID string) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, instanceFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, instanceFile)

				data, err := os.ReadFile(filepath.Clean(filepath.Join(tmpDir, instanceFile)))
				require.NoError(t, err)
				require.Contains(t, string(data), "instanceId: ")
			},
			wantErr: false,
			newID:   true,
		},
		{
			name: "empty_file",
			setupTestData: func(t *testing.T, tmpDir, instanceFile, instanceID string) {
				t.Helper()
				// create empty file
				_, err := os.Create(filepath.Clean(filepath.Join(tmpDir, instanceFile)))
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, instanceFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, instanceFile)

				data, err := os.ReadFile(filepath.Clean(filepath.Join(tmpDir, instanceFile)))
				require.NoError(t, err)
				require.Contains(t, string(data), "instanceId: ")
			},
			wantErr: false,
			newID:   true,
		},
		{
			name: "file_presents_single_line",
			setupTestData: func(t *testing.T, tmpDir, instanceFile, instanceID string) {
				t.Helper()
				err := os.WriteFile(filepath.Join(tmpDir, instanceFile), []byte(fmt.Sprintf("%s: %s", InstanceIDKey, instanceID)), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, instanceFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, instanceFile)
			},
			wantErr: false,
			newID:   false,
		},
		{
			name: "file_presents_multi_lines",
			setupTestData: func(t *testing.T, tmpDir, instanceFile, instanceID string) {
				t.Helper()
				data := fmt.Sprintf("PRODUCT_FAMILY_PS: 1\nPRODUCT_FAMILY_PXC: 1\nPRODUCT_FAMILY_PSMDB: 1\n%s: %s", InstanceIDKey, instanceID)
				err := os.WriteFile(filepath.Join(tmpDir, instanceFile), []byte(data), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, instanceFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, instanceFile)
			},
			wantErr: false,
			newID:   false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "test")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			instanceID := uuid.New().String()
			instanceFile := "telemetry_uuid"

			if tt.name == "non_existing_directory" {
				// make directory absent
				_ = os.RemoveAll(tmpDir)
			} else {
				tt.setupTestData(t, tmpDir, instanceFile, instanceID)
			}

			got, err := getInstanceID(filepath.Join(tmpDir, "telemetry_uuid"))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.newID {
					// in this case getInstanceID function generates new ID on its own.
					require.NotEmpty(t, got)
				} else {
					require.Equal(t, instanceID, got)
				}
			}
			tt.postCheckTestData(t, tmpDir, instanceFile)
		})
	}
}
