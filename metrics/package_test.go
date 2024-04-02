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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var osNames = []struct { //nolint:gochecknoglobals
	name           string
	osName         string
	expectedDebian bool
	expectedRhel   bool
}{
	{
		name:           "Enterprise Linux 8",
		osName:         "el8",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "Enterprise Linux 9",
		osName:         "el9",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "Ubuntu 22.04.3 LTS",
		osName:         "Ubuntu 22.04.3 LTS",
		expectedDebian: true,
		expectedRhel:   false,
	},
	{
		name:           "CentOS Linux 7 (Core)",
		osName:         "CentOS Linux 7 (Core)",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "Debian GNU/Linux 10 (buster)",
		osName:         "Debian GNU/Linux 10 (buster)",
		expectedDebian: true,
		expectedRhel:   false,
	},
	{
		name:           "Oracle Linux Server 8.9",
		osName:         "Oracle Linux Server 8.9",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "Amazon Linux 2",
		osName:         "Amazon Linux 2",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "CentOS Stream 8",
		osName:         "CentOS Stream 8",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "Rocky Linux 8.9 (Green Obsidian)",
		osName:         "Rocky Linux 8.9 (Green Obsidian)",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "Red Hat Enterprise Linux 8.9 (Ootpa)",
		osName:         "Red Hat Enterprise Linux 8.9 (Ootpa)",
		expectedDebian: false,
		expectedRhel:   true,
	},
	{
		name:           "AlmaLinux 8.9 (Midnight Oncilla)",
		osName:         "AlmaLinux 8.9 (Midnight Oncilla)",
		expectedDebian: false,
		expectedRhel:   true,
	},
}

func TestIsDebianFamily(t *testing.T) {
	t.Parallel()

	for _, tt := range osNames {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expectedDebian, isDebianFamily(tt.osName))
		})
	}
}

func TestIsRHELFamily(t *testing.T) {
	t.Parallel()

	for _, tt := range osNames {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expectedRhel, isRHELFamily(tt.osName))
		})
	}
}

func TestParseDpkgOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		packageNamePattern string
		dpkgOutput         []byte
		dpkgErr            error
		expectedPkgList    []*Package
		expectErr          error
	}{
		{
			name:               "pattern_percona_full_output",
			packageNamePattern: "percona-*",
			dpkgOutput: []byte(`ii |percona-backup-mongodb|2.3.1-1.jammy
ii |percona-mongodb-mongosh|2.1.1.jammy
ii |percona-mysql-router|8.2.0-1-1.jammy
ii |percona-mysql-shell:amd64|8.2.0-1-1.jammy
ii |percona-pg-stat-monitor16|1:2.0.4-2.jammy
iHR |percona-pgbouncer|1:1.22.0-1.jammy
ii |percona-postgresql-16|2:16.2-1.jammy
ii |percona-postgresql-16-pgaudit|1:16.0-1.jammy
iHR |percona-postgresql-16-wal2json|1:2.5-7.jammy
ii |percona-release|1.0-27.generic
ii |percona-server-client|8.2.0-1-1.jammy
un |percona-server-client-5.7|
ii |percona-server-mongodb|7.0.5-3.jammy
ii |percona-server-mongodb-mongos|7.0.5-3.jammy
un |percona-server-mongodb-mongos-pro|
un |percona-server-mongodb-pro|
ii |percona-server-mongodb-server|7.0.5-3.jammy
un |percona-server-mongodb-server-pro|
ii |percona-server-server|8.2.0-1-1.jammy
iHR |percona-toolkit|3.5.7-1.jammy
un |percona-xtrabackup|
ii |percona-xtrabackup-81|8.1.0-1-1.jammy
un |percona-xtradb-client-5.0|
un |percona-xtradb-server-5.0|
`),
			dpkgErr: nil,
			expectedPkgList: []*Package{
				{
					Name:    "percona-backup-mongodb",
					Version: "2.3.1-1",
				},
				{
					Name:    "percona-mongodb-mongosh",
					Version: "2.1.1",
				},
				{
					Name:    "percona-mysql-router",
					Version: "8.2.0-1-1",
				},
				{
					Name:    "percona-mysql-shell",
					Version: "8.2.0-1-1",
				},
				{
					Name:    "percona-pg-stat-monitor16",
					Version: "2.0.4-2",
				},
				{
					Name:    "percona-pgbouncer",
					Version: "1.22.0-1",
				},
				{
					Name:    "percona-postgresql-16",
					Version: "16.2-1",
				},
				{
					Name:    "percona-postgresql-16-pgaudit",
					Version: "16.0-1",
				},
				{
					Name:    "percona-postgresql-16-wal2json",
					Version: "2.5-7",
				},
				{
					Name:    "percona-release",
					Version: "1.0-27",
				},
				{
					Name:    "percona-server-client",
					Version: "8.2.0-1-1",
				},
				{
					Name:    "percona-server-mongodb",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-server-mongodb-mongos",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-server-mongodb-server",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-server-server",
					Version: "8.2.0-1-1",
				},
				{
					Name:    "percona-toolkit",
					Version: "3.5.7-1",
				},
				{
					Name:    "percona-xtrabackup-81",
					Version: "8.1.0-1-1",
				},
			},
			expectErr: nil,
		},
		{
			name:               "pattern_percona_proxysql_installed_full_output",
			packageNamePattern: "proxysql*",
			dpkgOutput: []byte(`ii |proxysql|1:1.5.5-1.2.jammy
iHR |proxysql2|2:2.5.5-1.2.jammy
`),
			dpkgErr: nil,
			expectedPkgList: []*Package{
				{
					Name:    "proxysql",
					Version: "1.5.5-1-2",
				},
				{
					Name:    "proxysql2",
					Version: "2.5.5-1-2",
				},
			},
			expectErr: nil,
		},
		{
			name:               "pattern_percona_pmm_installed_full_output",
			packageNamePattern: "pmm*",
			dpkgOutput: []byte(`un |pmm-client|
ii |pmm2-client|2.41.2-6.1.jammy
`),
			dpkgErr: nil,
			expectedPkgList: []*Package{
				{
					Name:    "pmm2-client",
					Version: "2.41.2-6-1",
				},
			},
			expectErr: nil,
		},
		{
			name:               "exact_external_installed_full_version_with_dfsg",
			packageNamePattern: "etcd",
			dpkgOutput:         []byte(`ii |etcd|3.3.25+dfsg-7ubuntu0.22.04.1`),
			dpkgErr:            nil,
			expectedPkgList: []*Package{
				{
					Name:    "etcd",
					Version: "3.3.25",
				},
			},
			expectErr: nil,
		},
		{
			name:               "exact_external_installed_full_version_with_epoch_with_dfsg",
			packageNamePattern: "etcd",
			dpkgOutput:         []byte(`ii |etcd|1:3.3.25+dfsg-7ubuntu0.22.04.1`),
			dpkgErr:            nil,
			expectedPkgList: []*Package{
				{
					Name:    "etcd",
					Version: "3.3.25",
				},
			},
			expectErr: nil,
		},
		{
			name:               "exact_external_installed_full_version_with_arch_with_epoch_with_dfsg",
			packageNamePattern: "etcd",
			dpkgOutput:         []byte(`ii |etcd:amd64|1:3.3.25+dfsg-7ubuntu0.22.04.1`),
			dpkgErr:            nil,
			expectedPkgList: []*Package{
				{
					Name:    "etcd",
					Version: "3.3.25",
				},
			},
			expectErr: nil,
		},
		{
			name:               "percona_not_installed",
			packageNamePattern: "percona-*",
			dpkgOutput:         []byte(`un |percona-xtrabackup-81|`),
			dpkgErr:            errors.New("no packages found matching percona-*"),
			expectedPkgList:    nil,
			expectErr:          errPackageNotFound,
		},
		{
			name:               "percona_not_found",
			packageNamePattern: "percona2-*",
			dpkgOutput:         []byte(`no packages found matching percona2-*`),
			dpkgErr:            errors.New("no packages found matching percona2-*"),
			expectedPkgList:    nil,
			expectErr:          errPackageNotFound,
		},
		{
			name:               "dpkg_error",
			packageNamePattern: "percona-*",
			dpkgOutput:         []byte(``),
			dpkgErr:            errors.New("dpkg-query: error while loading shared libraries: libapt-pkg.so.6.0: cannot open shared object file: No such file or directory"),
			expectedPkgList:    nil,
			expectErr:          errors.New("dpkg-query: error while loading shared libraries: libapt-pkg.so.6.0: cannot open shared object file: No such file or directory"),
		},
		{
			name:               "invalid_dpkg_output",
			packageNamePattern: "percona-xtrabackup-81",
			dpkgOutput:         []byte(`ii |percona-xtrabackup-81`),
			dpkgErr:            nil,
			expectedPkgList:    nil,
			expectErr:          errPackageNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, err := parseDebianOutput(tt.packageNamePattern, tt.dpkgOutput, tt.dpkgErr)
			if tt.expectErr != nil {
				require.ErrorAs(t, err, &tt.expectErr)
			}

			if tt.expectedPkgList != nil {
				require.Equal(t, tt.expectedPkgList, pkg)
			}
		})
	}
}

func TestParseRpmOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		packageNamePattern string
		rpmOutput          []byte
		rpmErr             error
		expectedPkgList    []*Package
		expectErr          error
	}{
		{
			name:               "pattern_percona_full_output",
			packageNamePattern: "percona-*",
			rpmOutput: []byte(`percona-server-server|8.0.36|28.1.el9
percona-mysql-shell|8.0.36|1.el9
percona-mongodb-mongosh|2.1.1|1.el9
percona-server-mongodb-server|7.0.5|3.el9
percona-postgresql16|16.2|2.el9
percona-postgresql16-server|16.2|2.el9
percona-pg_stat_monitor16|2.0.4|2.el9
percona-pgaudit16|16.0|2.el9
percona-wal2json16|2.5|2.el9
percona-pgbouncer|1.22.0|1.el9
percona-server-mongodb|7.0.5|3.el9
percona-xtrabackup-81|8.1.0|1.1.el9
percona-toolkit|3.5.7|1.el9
percona-backup-mongodb|2.4.1.el9|
`),
			rpmErr: nil,
			expectedPkgList: []*Package{
				{
					Name:    "percona-server-server",
					Version: "8.0.36-28-1",
				},
				{
					Name:    "percona-mysql-shell",
					Version: "8.0.36-1",
				},
				{
					Name:    "percona-mongodb-mongosh",
					Version: "2.1.1-1",
				},
				{
					Name:    "percona-server-mongodb-server",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-postgresql16",
					Version: "16.2-2",
				},
				{
					Name:    "percona-postgresql16-server",
					Version: "16.2-2",
				},
				{
					Name:    "percona-pg_stat_monitor16",
					Version: "2.0.4-2",
				},
				{
					Name:    "percona-pgaudit16",
					Version: "16.0-2",
				},
				{
					Name:    "percona-wal2json16",
					Version: "2.5-2",
				},
				{
					Name:    "percona-pgbouncer",
					Version: "1.22.0-1",
				},
				{
					Name:    "percona-server-mongodb",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-xtrabackup-81",
					Version: "8.1.0-1-1",
				},
				{
					Name:    "percona-toolkit",
					Version: "3.5.7-1",
				},
				{
					Name:    "percona-backup-mongodb",
					Version: "2.4.1",
				},
			},
			expectErr: nil,
		},
		{
			name:               "pattern_percona_proxysql_installed",
			packageNamePattern: "proxysql*",
			rpmOutput: []byte(`proxysql|1.5.5|1.2.el9
proxysql2|2.5.5|1.2.el9`),
			rpmErr: nil,
			expectedPkgList: []*Package{
				{
					Name:    "proxysql",
					Version: "1.5.5-1-2",
				},
				{
					Name:    "proxysql2",
					Version: "2.5.5-1-2",
				},
			},
			expectErr: nil,
		},
		{
			name:               "pattern_percona_pmm_installed",
			packageNamePattern: "pmm*",
			rpmOutput:          []byte(`pmm2-client|2.41.2|6.1.el9`),
			rpmErr:             nil,
			expectedPkgList: []*Package{
				{
					Name:    "pmm2-client",
					Version: "2.41.2-6-1",
				},
			},
			expectErr: nil,
		},
		{
			name:               "exact_external_installed",
			packageNamePattern: "etcd",
			rpmOutput:          []byte(`etcd|3.5.12|1.el8`),
			rpmErr:             nil,
			expectedPkgList: []*Package{
				{
					Name:    "etcd",
					Version: "3.5.12",
				},
			},
			expectErr: nil,
		},
		{
			name:               "percona_not_installed",
			packageNamePattern: "percona-*",
			rpmOutput:          []byte(``),
			rpmErr:             nil,
			expectedPkgList:    nil,
			expectErr:          errPackageNotFound,
		},
		{
			name:               "external_not_installed",
			packageNamePattern: "etcd*",
			rpmOutput:          []byte(``),
			rpmErr:             nil,
			expectedPkgList:    nil,
			expectErr:          errPackageNotFound,
		},
		{
			name:               "rpm_error",
			packageNamePattern: "percona-*",
			rpmOutput:          []byte(``),
			rpmErr:             errors.New("rpm: --test may only be specified during package installation and erasure"),
			expectedPkgList:    nil,
			expectErr:          errors.New("rpm: --test may only be specified during package installation and erasure"),
		},
		{
			name:               "invalid_rpm_output",
			packageNamePattern: "percona-xtrabackup-81",
			rpmOutput:          []byte(`percona-xtrabackup-81|`),
			rpmErr:             nil,
			expectedPkgList:    nil,
			expectErr:          errPackageNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, err := parseRhelOutput(tt.packageNamePattern, tt.rpmOutput, tt.rpmErr)
			if tt.expectErr != nil {
				require.ErrorAs(t, err, &tt.expectErr)
			}

			if tt.expectedPkgList != nil {
				require.Equal(t, tt.expectedPkgList, pkg)
			}
		})
	}
}

func TestDpkgRpmEqualOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		packageNamePattern  string
		dpkgOutput          []byte
		dpkgErr             error
		expectedDpkgErr     error
		expectedDpkgPkgList []*Package
		//
		rpmOutput          []byte
		rpmErr             error
		expectedRpmErr     error
		expectedRpmPkgList []*Package
	}{
		{
			name:               "pattern_percona_full_output",
			packageNamePattern: "percona-*",
			dpkgOutput: []byte(`ii |percona-server-server|8.0.36-28-1.jammy
ii |percona-mysql-shell:amd64|8.0.36-1.jammy
ii |percona-mongodb-mongosh|2.1.1-1.jammy
ii |percona-server-mongodb-server|7.0.5-3.jammy
ii |percona-server-mongodb|7.0.5-3.jammy
ii |percona-backup-mongodb|2.4.1-1.jammy
ii |percona-xtrabackup-81|8.1.0-1-1.jammy
iHR |percona-toolkit|3.5.7-1.jammy
`),
			rpmOutput: []byte(`percona-server-server|8.0.36|28.1.el9
percona-mysql-shell|8.0.36|1.el9
percona-mongodb-mongosh|2.1.1|1.el9
percona-server-mongodb-server|7.0.5|3.el9
percona-server-mongodb|7.0.5|3.el9
percona-backup-mongodb|2.4.1|1.el9
percona-xtrabackup-81|8.1.0|1.1.el9
percona-toolkit|3.5.7|1.el9
`),
			expectedDpkgPkgList: []*Package{
				{
					Name:    "percona-server-server",
					Version: "8.0.36-28-1",
				},
				{
					Name:    "percona-mysql-shell",
					Version: "8.0.36-1",
				},
				{
					Name:    "percona-mongodb-mongosh",
					Version: "2.1.1-1",
				},
				{
					Name:    "percona-server-mongodb-server",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-server-mongodb",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-backup-mongodb",
					Version: "2.4.1-1",
				},
				{
					Name:    "percona-xtrabackup-81",
					Version: "8.1.0-1-1",
				},
				{
					Name:    "percona-toolkit",
					Version: "3.5.7-1",
				},
			},
			expectedRpmPkgList: []*Package{
				{
					Name:    "percona-server-server",
					Version: "8.0.36-28-1",
				},
				{
					Name:    "percona-mysql-shell",
					Version: "8.0.36-1",
				},
				{
					Name:    "percona-mongodb-mongosh",
					Version: "2.1.1-1",
				},
				{
					Name:    "percona-server-mongodb-server",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-server-mongodb",
					Version: "7.0.5-3",
				},
				{
					Name:    "percona-backup-mongodb",
					Version: "2.4.1-1",
				},
				{
					Name:    "percona-xtrabackup-81",
					Version: "8.1.0-1-1",
				},
				{
					Name:    "percona-toolkit",
					Version: "3.5.7-1",
				},
			},
		},
		{
			name:               "pattern_external_full_output",
			packageNamePattern: "etcd*",
			dpkgOutput:         []byte(`ii |etcd:amd64|1:3.3.25+dfsg-7ubuntu0.22.04.1`),
			rpmOutput:          []byte(`etcd|3.3.25|1.el8`),
			expectedDpkgPkgList: []*Package{
				{
					Name:    "etcd",
					Version: "3.3.25",
				},
			},
			expectedRpmPkgList: []*Package{
				{
					Name:    "etcd",
					Version: "3.3.25",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// dpkg
			dpkgPkgList, err := parseDebianOutput(tt.packageNamePattern, tt.dpkgOutput, tt.dpkgErr)
			if tt.expectedDpkgErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorAs(t, err, &tt.expectedDpkgErr)
			}

			if tt.expectedDpkgPkgList != nil {
				require.Equal(t, tt.expectedDpkgPkgList, dpkgPkgList)
			}

			// rpm
			rpmPkgList, err := parseRhelOutput(tt.packageNamePattern, tt.rpmOutput, tt.expectedRpmErr)
			if tt.expectedRpmErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorAs(t, err, &tt.expectedRpmErr)
			}

			if tt.expectedDpkgPkgList != nil {
				require.Equal(t, tt.expectedDpkgPkgList, rpmPkgList)
			}

			if tt.expectedDpkgPkgList != nil && tt.expectedRpmPkgList != nil {
				require.Equal(t, dpkgPkgList, rpmPkgList)
			}
		})
	}
}
