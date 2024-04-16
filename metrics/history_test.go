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
	"time"

	"github.com/google/uuid"
	platformReporter "github.com/percona-platform/saas/gen/telemetry/generic"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWriteMetricsToHistory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		request           *platformReporter.ReportRequest
		setupTestData     func(t *testing.T, tmpDir, token string, currTime time.Time)                                                   // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir, historyFile, token string, currTime time.Time, req *platformReporter.ReportRequest) // Post CleanupMetricsHistory function validation
		wantErr           bool
	}{
		{
			name: "non_existing_directory",
			setupTestData: func(t *testing.T, tmpDir, _ string, _ time.Time) {
				t.Helper()
				// make directory absent
				_ = os.RemoveAll(tmpDir)
			},
			postCheckTestData: func(t *testing.T, tmpDir, _, _ string, _ time.Time, _ *platformReporter.ReportRequest) {
				t.Helper()
				// directory shall not be created
				_, err := os.Stat(tmpDir)
				require.ErrorIs(t, err, os.ErrNotExist)
			},
			request: &platformReporter.ReportRequest{},
			wantErr: true,
		},
		{
			name: "empty_request_empty_directory",
			setupTestData: func(t *testing.T, _, _ string, _ time.Time) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, _, _ string, _ time.Time, _ *platformReporter.ReportRequest) {
				t.Helper()
				// no new files shall be created
				checkDirectoryContentCount(t, tmpDir, 0)
			},
			request: &platformReporter.ReportRequest{},
			wantErr: true,
		},
		{
			name: "empty_request_non_empty_directory",
			setupTestData: func(t *testing.T, tmpDir, token string, currTime time.Time) {
				t.Helper()
				writeTempFiles(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
			},
			postCheckTestData: func(t *testing.T, tmpDir, historyFile, token string, currTime time.Time, _ *platformReporter.ReportRequest) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 3)

				// all these files shall be kept in directory
				checkFilesExist(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
				// history file shall not be created
				checkFilesAbsent(t, tmpDir, historyFile)
			},
			request: &platformReporter.ReportRequest{},
			wantErr: true,
		},
		{
			name: "no_request_reports_empty_directory",
			setupTestData: func(t *testing.T, _, _ string, _ time.Time) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, _, _ string, _ time.Time, _ *platformReporter.ReportRequest) {
				t.Helper()
				// no new files shall be created
				checkDirectoryContentCount(t, tmpDir, 0)
			},
			request: &platformReporter.ReportRequest{Reports: []*platformReporter.GenericReport{}},
			wantErr: true,
		},
		{
			name: "no_request_reports_non_empty_directory",
			setupTestData: func(t *testing.T, tmpDir, token string, currTime time.Time) {
				t.Helper()
				writeTempFiles(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
			},
			postCheckTestData: func(t *testing.T, tmpDir, historyFile, token string, currTime time.Time, _ *platformReporter.ReportRequest) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 3)

				// all these files shall be kept in directory
				checkFilesExist(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
				// history file shall not be created
				checkFilesAbsent(t, tmpDir, historyFile)
			},
			request: &platformReporter.ReportRequest{Reports: []*platformReporter.GenericReport{}},
			wantErr: true,
		},
		{
			name: "single_report_empty_directory",
			setupTestData: func(t *testing.T, _, _ string, _ time.Time) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir, historyFile, _ string, _ time.Time, req *platformReporter.ReportRequest) {
				t.Helper()
				// only one file shall be created
				checkDirectoryContentCount(t, tmpDir, 1)

				checkFilesExist(t, tmpDir, historyFile)

				// Verify history file content was written successfully.
				checkHistoryFileContent(t, tmpDir, historyFile, req)
			},
			request: &platformReporter.ReportRequest{Reports: []*platformReporter.GenericReport{{
				Id:            uuid.New().String(),
				CreateTime:    timestamppb.New(time.Now()),
				InstanceId:    uuid.New().String(),
				ProductFamily: platformReporter.ProductFamily_PRODUCT_FAMILY_PS,
				Metrics: []*platformReporter.GenericReport_Metric{
					{Key: "test_metric_1", Value: "test_value_1"},
					{Key: "test_metric_2", Value: "[\"file://component_percona_telemetry\"]"},
				},
			}}},
			wantErr: false,
		},
		{
			name: "single_report_non_empty_directory",
			setupTestData: func(t *testing.T, tmpDir, token string, currTime time.Time) {
				t.Helper()
				writeTempFiles(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
			},
			postCheckTestData: func(t *testing.T, tmpDir, historyFile, token string, currTime time.Time, req *platformReporter.ReportRequest) {
				t.Helper()
				// only one file shall be created
				checkDirectoryContentCount(t, tmpDir, 4)

				// all these files shall be kept in directory
				checkFilesExist(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token),
					historyFile)

				// Verify the file was written successfully.
				checkHistoryFileContent(t, tmpDir, historyFile, req)
			},
			request: &platformReporter.ReportRequest{Reports: []*platformReporter.GenericReport{{
				Id:            uuid.New().String(),
				CreateTime:    timestamppb.New(time.Now()),
				InstanceId:    uuid.New().String(),
				ProductFamily: platformReporter.ProductFamily_PRODUCT_FAMILY_PS,
				Metrics: []*platformReporter.GenericReport_Metric{
					{Key: "test_metric_1", Value: "test_value_1"},
					{Key: "test_metric_2", Value: "[\"file://component_percona_telemetry\"]"},
				},
			}}},
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, err := os.MkdirTemp("", "test-history")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			currTime := time.Now()
			token := uuid.New().String()
			historyFile := fmt.Sprintf("%d-history.json", currTime.Unix())
			tt.setupTestData(t, tmpDir, token, currTime)

			err = WriteMetricsToHistory(filepath.Join(tmpDir, historyFile), tt.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			tt.postCheckTestData(t, tmpDir, historyFile, token, currTime, tt.request)
		})
	}
}

func TestCleanupMetricsHistory(t *testing.T) {
	t.Parallel()

	currTime, token := time.Now(), uuid.New().String()
	testCases := []struct {
		name              string
		setupTestData     func(t *testing.T, tmpDir string) // Setups necessary data for the test
		postCheckTestData func(t *testing.T, tmpDir string) // Post CleanupMetricsHistory function validation
		keepInterval      int                               // Input to CleanupMetricsHistory function
		wantErr           bool                              // true if you expect an error in CleanupMetricsHistory function
	}{
		{
			name: "all_files_within_keep_interval",
			setupTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				writeTempFiles(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
			},
			postCheckTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				// all these files shall be kept in directory
				checkDirectoryContentCount(t, tmpDir, 3)
				checkFilesExist(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-10*time.Minute)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-20*time.Minute)).Unix(), token))
			},
			keepInterval: 7200,
			wantErr:      false,
		},
		{
			name: "some_files_beyond_keep_interval",
			setupTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				writeTempFiles(t, tmpDir,
					fmt.Sprintf("%d-%s.json", currTime.Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-2*time.Hour)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-24*time.Hour)).Unix(), token))
			},
			postCheckTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				// all these files shall be kept in directory
				checkDirectoryContentCount(t, tmpDir, 1)
				checkFilesExist(t, tmpDir, fmt.Sprintf("%d-%s.json", currTime.Unix(), token))
				// all these files shall be removed from directory
				checkFilesAbsent(t, tmpDir,
					fmt.Sprintf("%d-%s.json", (currTime.Add(-2*time.Hour)).Unix(), token),
					fmt.Sprintf("%d-%s.json", (currTime.Add(-24*time.Hour)).Unix(), token))
			},
			keepInterval: 3600,
			wantErr:      false,
		},
		{
			name: "empty_directory",
			setupTestData: func(t *testing.T, _ string) {
				t.Helper()
			},
			postCheckTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				checkDirectoryContentCount(t, tmpDir, 0)
			},
			keepInterval: 3600,
			wantErr:      false,
		},
		{
			name: "non_existing_directory",
			setupTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				// make directory absent
				_ = os.RemoveAll(tmpDir)
			},
			postCheckTestData: func(t *testing.T, tmpDir string) {
				t.Helper()
				// directory shall not be created
				_, err := os.Stat(tmpDir)
				require.ErrorIs(t, err, os.ErrNotExist)
			},
			keepInterval: 3600,
			wantErr:      true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, err := os.MkdirTemp("", "test-history")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			tt.setupTestData(t, tmpDir)

			err = CleanupMetricsHistory(tmpDir, tt.keepInterval)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			tt.postCheckTestData(t, tmpDir)
		})
	}
}
