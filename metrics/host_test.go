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

func TestGetInstanceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupTestData     func(t *testing.T, tmpDir, instanceFile, instanceID string) // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir, instanceFile string)             // Post function validation/cleanup
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
			newID: true,
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
			newID: true,
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
			newID: true,
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
			newID: false,
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
			newID: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

			got := getInstanceID(filepath.Join(tmpDir, "telemetry_uuid"))
			if tt.newID {
				// in this case getInstanceID function generates new ID on its own.
				require.NotEmpty(t, got)
			} else {
				require.Equal(t, instanceID, got)
			}
			tt.postCheckTestData(t, tmpDir, instanceFile)
		})
	}
}

// TestReadOSReleaseFile tests the function readOSReleaseFile.
func TestReadOSReleaseFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupTestData     func(t *testing.T, tmpDir, releaseFile string) // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir, releaseFile string) // Post function validation/cleanup
		want              string
	}{
		{
			name: "file_absent",
			setupTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 0)
				checkFilesAbsent(t, tmpDir, releaseFile)
			},
			want: unknownOS,
		},
		{
			name: "file_exists",
			setupTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				fileContent := `NAME="Oracle Linux Server"
VERSION="9.2"
ID="ol"
ID_LIKE="fedora"
VARIANT="Server"
VARIANT_ID="server"
VERSION_ID="9.2"
PLATFORM_ID="platform:el9"
PRETTY_NAME="Oracle Linux Server 9.2"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:oracle:linux:9:2:server"
HOME_URL="https://linux.oracle.com/"
BUG_REPORT_URL="https://github.com/oracle/oracle-linux"

ORACLE_BUGZILLA_PRODUCT="Oracle Linux 9"
ORACLE_BUGZILLA_PRODUCT_VERSION=9.2
ORACLE_SUPPORT_PRODUCT="Oracle Linux"
ORACLE_SUPPORT_PRODUCT_VERSION=9.2
`
				err := os.WriteFile(filepath.Join(tmpDir, releaseFile), []byte(fileContent), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, releaseFile)
			},
			want: "Oracle Linux Server 9.2",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, err := os.MkdirTemp("", "testOS")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			releaseFile := "os-release"
			tt.setupTestData(t, tmpDir, releaseFile)
			require.Equal(t, tt.want, readOSReleaseFile(filepath.Join(tmpDir, releaseFile)))
			tt.postCheckTestData(t, tmpDir, releaseFile)
		})
	}
}

// TestReadSystemReleaseFile tests the function readSystemReleaseFile.
func TestReadSystemReleaseFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupTestData     func(t *testing.T, tmpDir, releaseFile string) // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir, releaseFile string) // Post function validation/cleanup
		want              string
	}{
		{
			name: "file_absent",
			setupTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 0)
				checkFilesAbsent(t, tmpDir, releaseFile)
			},
			want: unknownOS,
		},
		{
			name: "system_format",
			setupTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				fileContent := "Oracle Linux Server release 9.2"
				err := os.WriteFile(filepath.Join(tmpDir, releaseFile), []byte(fileContent), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, releaseFile)
			},
			want: "Oracle Linux Server release 9.2",
		},
		{
			name: "redhat_format",
			setupTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				fileContent := "Red Hat Enterprise Linux release 9.2 (Plow)"
				err := os.WriteFile(filepath.Join(tmpDir, releaseFile), []byte(fileContent), 0o600)
				require.NoError(t, err)
			},
			postCheckTestData: func(t *testing.T, tmpDir, releaseFile string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, releaseFile)
			},
			want: "Red Hat Enterprise Linux release 9.2 (Plow)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, err := os.MkdirTemp("", "testOS")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			releaseFile := "system-release"
			tt.setupTestData(t, tmpDir, releaseFile)
			require.Equal(t, tt.want, readSystemReleaseFile(filepath.Join(tmpDir, releaseFile)))
			tt.postCheckTestData(t, tmpDir, releaseFile)
		})
	}
}

func TestGetDeploymentInfo(t *testing.T) { //nolint:paralleltest
	testCases := []struct {
		name          string
		setupTestData func(t *testing.T)
		expected      string
	}{
		{
			name: "no_env_defined",
			setupTestData: func(t *testing.T) {
				t.Helper()
			},
			expected: deploymentPackage,
		},
		{
			name: "env_defined_empty",
			setupTestData: func(t *testing.T) {
				t.Helper()

				t.Setenv(perconaDockerEnv, "")
			},
			expected: deploymentDocker,
		},
		{
			name: "env_defined",
			setupTestData: func(t *testing.T) {
				t.Helper()

				t.Setenv(perconaDockerEnv, "")
			},
			expected: deploymentDocker,
		},
	}

	for _, tt := range testCases { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			tt.setupTestData(t)
			got := getDeploymentInfo()
			require.Equal(t, tt.expected, got)
		})
	}
}
