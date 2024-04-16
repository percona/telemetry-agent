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
	"testing"

	"github.com/stretchr/testify/require"
)

var osNames = []struct { //nolint:gochecknoglobals
	name     string
	osName   string
	expected int
}{
	{
		name:     "Enterprise Linux 8",
		osName:   "el8",
		expected: distroFamilyRhel,
	},
	{
		name:     "Enterprise Linux 9",
		osName:   "el9",
		expected: distroFamilyRhel,
	},
	{
		name:     "Ubuntu 22.04.3 LTS",
		osName:   "Ubuntu 22.04.3 LTS",
		expected: distroFamilyDebian,
	},
	{
		name:     "CentOS Linux 7 (Core)",
		osName:   "CentOS Linux 7 (Core)",
		expected: distroFamilyRhel,
	},
	{
		name:     "Debian GNU/Linux 10 (buster)",
		osName:   "Debian GNU/Linux 10 (buster)",
		expected: distroFamilyDebian,
	},
	{
		name:     "Oracle Linux Server 8.9",
		osName:   "Oracle Linux Server 8.9",
		expected: distroFamilyRhel,
	},
	{
		name:     "Amazon Linux 2",
		osName:   "Amazon Linux 2",
		expected: distroFamilyRhel,
	},
	{
		name:     "CentOS Stream 8",
		osName:   "CentOS Stream 8",
		expected: distroFamilyRhel,
	},
	{
		name:     "Rocky Linux 8.9 (Green Obsidian)",
		osName:   "Rocky Linux 8.9 (Green Obsidian)",
		expected: distroFamilyRhel,
	},
	{
		name:     "Red Hat Enterprise Linux 8.9 (Ootpa)",
		osName:   "Red Hat Enterprise Linux 8.9 (Ootpa)",
		expected: distroFamilyRhel,
	},
	{
		name:     "AlmaLinux 8.9 (Midnight Oncilla)",
		osName:   "AlmaLinux 8.9 (Midnight Oncilla)",
		expected: distroFamilyRhel,
	},
	{
		name:     "MacOS",
		osName:   "Darwin",
		expected: distroFamilyUnknown,
	},
}

func TestIsPerconaPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{
			name:     "empty_pattern",
			pattern:  "",
			expected: false,
		},
		{
			name:     "common_percona_package_percona",
			pattern:  "percona-*",
			expected: true,
		},
		{
			name:     "common_percona_package_proxysql",
			pattern:  "proxysql*",
			expected: true,
		},
		{
			name:     "common_percona_package_pmm",
			pattern:  "pmm*",
			expected: true,
		},
		{
			name:     "common_external_package_etcd",
			pattern:  "etcd",
			expected: false,
		},
		{
			name:     "common_external_package_haproxy",
			pattern:  "haproxy",
			expected: false,
		},
		{
			name:     "common_external_package_patroni",
			pattern:  "patroni",
			expected: false,
		},
		{
			name:     "common_external_package_pg",
			pattern:  "pg*",
			expected: false,
		},
		{
			name:     "common_external_package_postgis",
			pattern:  "postgis",
			expected: false,
		},
		{
			name:     "common_external_package_wal2json",
			pattern:  "wal2json",
			expected: false,
		},
		{
			name:     "debian_percona_package_percona",
			pattern:  "Percona-*",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isPerconaPackage(tt.pattern)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDebianRhelEqualOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                        string
		isPerconaPackage            bool
		debianPackageOutput         []byte
		debianPackageErr            error
		debianPackageExpectedErr    error
		debianRepositoryOutput      [][]byte
		debianRepositoryErr         error
		debianRepositoryExpectedErr error
		rhelPackageOutput           []byte
		rhelPackageErr              error
		rhelExpectedErr             error
		expectedPackageList         []*Package
	}{
		{
			name:             "pattern_percona_full_output",
			isPerconaPackage: isPerconaPackage("percona-*"),
			debianPackageOutput: []byte(`ii |percona-server-server|8.0.36-28-1.jammy
ii |percona-server-mongodb-server|7.0.5-3.jammy
ii |percona-backup-mongodb|2.4.1-1.jammy
`),
			debianPackageErr:         nil,
			debianPackageExpectedErr: nil,
			debianRepositoryOutput: [][]byte{
				[]byte(`percona-server-server:
Installed: 8.0.36-28-1.jammy
Candidate: 8.0.36-28-1.jammy
Version table:
*** 8.0.36-28-1.jammy 500
        500 http://repo.percona.com/ps-80/apt jammy/main amd64 Packages
        100 /var/lib/dpkg/status
    8.0.35-27-1.jammy 500
        500 http://repo.percona.com/ps-80/apt jammy/main amd64 Packages
    8.0.34-26-1.jammy 500
        500 http://repo.percona.com/ps-80/apt jammy/main amd64 Packages
`),
				[]byte(`percona-server-mongodb-server:
Installed: 7.0.5-3.jammy
Candidate: 7.0.5-3.jammy
Version table:
*** 7.0.5-3.jammy 500
		500 http://repo.percona.com/pdmdb-7.0/apt jammy/main amd64 Packages
		100 /var/lib/dpkg/status
	7.0.4-2.jammy 500
		500 http://repo.percona.com/pdmdb-7.0/apt jammy/main amd64 Packages
`),
				[]byte(`percona-backup-mongodb:
Installed: 2.4.1-1.jammy
Candidate: 2.4.1-1.jammy
Version table:
*** 2.4.1-1.jammy 500
		500 http://repo.percona.com/pbm/apt jammy/main amd64 Packages
		500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
		100 /var/lib/dpkg/status
	2.4.0-1.jammy 500
		500 http://repo.percona.com/pbm/apt jammy/main amd64 Packages
		500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
`),
			},
			debianRepositoryErr:         nil,
			debianRepositoryExpectedErr: nil,
			rhelPackageOutput: []byte(`percona-server-server|8.0.36|28.1.el9|ps-80-release-x86_64
percona-server-mongodb-server|7.0.5|3.el9|pdmdb-7.0-release-x86_64
percona-backup-mongodb|2.4.1|1.el9|pbm-release-x86_64
`),
			rhelPackageErr: nil,
			expectedPackageList: []*Package{
				{
					Name:    "percona-server-server",
					Version: "8.0.36-28-1",
					Repository: PackageRepository{
						Name:      "ps-80",
						Component: "release",
					},
				},
				{
					Name:    "percona-server-mongodb-server",
					Version: "7.0.5-3",
					Repository: PackageRepository{
						Name:      "pdmdb-7.0",
						Component: "release",
					},
				},
				{
					Name:    "percona-backup-mongodb",
					Version: "2.4.1-1",
					Repository: PackageRepository{
						Name:      "pbm",
						Component: "release",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// dpkg
			debianPkgList, err := parseDebianPackageOutput(tt.debianPackageOutput, tt.debianPackageErr, tt.isPerconaPackage)
			if tt.debianPackageExpectedErr == nil {
				require.NoError(t, err)
				require.NotNil(t, debianPkgList)
			} else {
				require.ErrorIs(t, err, tt.debianPackageExpectedErr)
				require.Nil(t, debianPkgList)
			}

			for i, pkg := range debianPkgList {
				debianPkgRepository, repoErr := parseDebianRepositoryOutput(tt.debianRepositoryOutput[i], tt.debianRepositoryErr, tt.isPerconaPackage)
				if tt.debianRepositoryExpectedErr == nil {
					require.NoError(t, repoErr)
					require.NotNil(t, debianPkgRepository)

					pkg.Repository = *debianPkgRepository
				} else {
					require.ErrorIs(t, repoErr, tt.debianRepositoryExpectedErr)
					require.Nil(t, debianPkgRepository)
				}
			}

			require.Equal(t, tt.expectedPackageList, debianPkgList)

			// rpm
			rhelPkgList, err := parseRhelPackageOutput(tt.rhelPackageOutput, tt.rhelExpectedErr, tt.isPerconaPackage)
			if tt.rhelExpectedErr == nil {
				require.NoError(t, err)
				require.NotNil(t, rhelPkgList)
			} else {
				require.ErrorIs(t, err, tt.rhelExpectedErr)
				require.Nil(t, rhelPkgList)
			}

			require.Equal(t, tt.expectedPackageList, rhelPkgList)
		})
	}
}
