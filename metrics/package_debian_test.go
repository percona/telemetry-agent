package metrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsDebianFamily(t *testing.T) {
	t.Parallel()

	for _, tt := range osNames {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, getDistroFamily(tt.osName))
		})
	}
}

func TestParseDebianPackageOutput(t *testing.T) {
	t.Parallel()

	dpkgErr := errors.New("dpkg-query: error while loading shared libraries: libapt-pkg.so.6.0: cannot open shared object file: No such file or directory")
	tests := []struct {
		name                string
		isPerconaPackage    bool
		packageOutput       []byte
		packageErr          error
		expectedPackageList []*Package
		expectErr           error
	}{
		{
			name:             "pattern_percona_full_output",
			isPerconaPackage: isPerconaPackage("percona-*"),
			packageOutput: []byte(`ii |percona-backup-mongodb|2.3.1-1.jammy
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
			packageErr: nil,
			expectedPackageList: []*Package{
				{
					Name:       "percona-backup-mongodb",
					Version:    "2.3.1-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-mongodb-mongosh",
					Version:    "2.1.1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-mysql-router",
					Version:    "8.2.0-1-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-mysql-shell",
					Version:    "8.2.0-1-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-pg-stat-monitor16",
					Version:    "2.0.4-2",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-pgbouncer",
					Version:    "1.22.0-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-postgresql-16",
					Version:    "16.2-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-postgresql-16-pgaudit",
					Version:    "16.0-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-postgresql-16-wal2json",
					Version:    "2.5-7",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-release",
					Version:    "1.0-27",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-server-client",
					Version:    "8.2.0-1-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-server-mongodb",
					Version:    "7.0.5-3",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-server-mongodb-mongos",
					Version:    "7.0.5-3",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-server-mongodb-server",
					Version:    "7.0.5-3",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-server-server",
					Version:    "8.2.0-1-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-toolkit",
					Version:    "3.5.7-1",
					Repository: PackageRepository{},
				},
				{
					Name:       "percona-xtrabackup-81",
					Version:    "8.1.0-1-1",
					Repository: PackageRepository{},
				},
			},
			expectErr: nil,
		},
		{
			name:             "pattern_percona_proxysql_installed_full_output",
			isPerconaPackage: isPerconaPackage("proxysql*"),
			packageOutput: []byte(`ii |proxysql|1:1.5.5-1.2.jammy
iHR |proxysql2|2:2.5.5-1.2.jammy
`),
			packageErr: nil,
			expectedPackageList: []*Package{
				{
					Name:       "proxysql",
					Version:    "1.5.5-1-2",
					Repository: PackageRepository{},
				},
				{
					Name:       "proxysql2",
					Version:    "2.5.5-1-2",
					Repository: PackageRepository{},
				},
			},
			expectErr: nil,
		},
		{
			name:             "pattern_percona_pmm_installed_full_output",
			isPerconaPackage: isPerconaPackage("pmm*"),
			packageOutput: []byte(`un |pmm-client|
ii |pmm2-client|2.41.2-6.1.jammy
`),
			packageErr: nil,
			expectedPackageList: []*Package{
				{
					Name:       "pmm2-client",
					Version:    "2.41.2-6-1",
					Repository: PackageRepository{},
				},
			},
			expectErr: nil,
		},
		{
			name:             "exact_external_installed_full_version_with_dfsg",
			isPerconaPackage: isPerconaPackage("etcd"),
			packageOutput:    []byte(`ii |etcd|3.3.25+dfsg-7ubuntu0.22.04.1`),
			packageErr:       nil,
			expectedPackageList: []*Package{
				{
					Name:       "etcd",
					Version:    "3.3.25",
					Repository: PackageRepository{},
				},
			},
			expectErr: nil,
		},
		{
			name:             "exact_external_installed_full_version_with_epoch_with_dfsg",
			isPerconaPackage: isPerconaPackage("etcd"),
			packageOutput:    []byte(`ii |etcd|1:3.3.25+dfsg-7ubuntu0.22.04.1`),
			packageErr:       nil,
			expectedPackageList: []*Package{
				{
					Name:       "etcd",
					Version:    "3.3.25",
					Repository: PackageRepository{},
				},
			},
			expectErr: nil,
		},
		{
			name:             "exact_external_installed_full_version_with_arch_with_epoch_with_dfsg",
			isPerconaPackage: isPerconaPackage("etcd"),
			packageOutput:    []byte(`ii |etcd:amd64|1:3.3.25+dfsg-7ubuntu0.22.04.1`),
			packageErr:       nil,
			expectedPackageList: []*Package{
				{
					Name:       "etcd",
					Version:    "3.3.25",
					Repository: PackageRepository{},
				},
			},
			expectErr: nil,
		},
		{
			name:                "percona_not_installed",
			isPerconaPackage:    isPerconaPackage("percona-*"),
			packageOutput:       []byte(`un |percona-xtrabackup-81|`),
			packageErr:          errors.New("dpkg-query: no packages found matching percona-*"),
			expectedPackageList: nil,
			expectErr:           errPackageNotFound,
		},
		{
			name:                "percona_not_found",
			isPerconaPackage:    isPerconaPackage("percona-*"),
			packageOutput:       []byte(`no packages found matching percona2-*`),
			packageErr:          errors.New("dpkg-query:  no packages found matching percona2-*"),
			expectedPackageList: nil,
			expectErr:           errPackageNotFound,
		},
		{
			name:                "dpkg_error",
			isPerconaPackage:    isPerconaPackage("percona-*"),
			packageOutput:       []byte(``),
			packageErr:          dpkgErr,
			expectedPackageList: nil,
			expectErr:           dpkgErr,
		},
		{
			name:                "invalid_dpkg_output",
			isPerconaPackage:    isPerconaPackage("percona-*"),
			packageOutput:       []byte(`ii |percona-xtrabackup-81`),
			packageErr:          nil,
			expectedPackageList: nil,
			expectErr:           errPackageNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, err := parseDebianPackageOutput(tt.packageOutput, tt.packageErr, tt.isPerconaPackage)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			}

			if tt.expectedPackageList != nil {
				require.Equal(t, tt.expectedPackageList, pkg)
			}
		})
	}
}

func TestParseDebianRepositoryOutput(t *testing.T) {
	t.Parallel()
	repositoryErr := errors.New("command line option 'r' [from -res] is not understood in combination with the other options")

	tests := []struct {
		name               string
		isPerconaPackage   bool
		repositoryOutput   []byte
		repositoryErr      error
		expectedRepository *PackageRepository
		expectErr          error
	}{
		{
			name:             "percona-backup-mongodb_installed",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-backup-mongodb:
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
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "pbm",
				Component: "release",
			},
			expectErr: nil,
		},
		{
			name:             "percona-server-server_installed_release",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-server-server:
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
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "ps-80",
				Component: "release",
			},
			expectErr: nil,
		},
		{
			name:             "percona-server-server_installed_testing",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-server-server:
  Installed: 8.0.36-28-1.jammy
  Candidate: 8.0.36-28-1.jammy
  Version table:
 *** 8.0.36-28-1.jammy 500
        500 http://repo.percona.com/ps-80/apt jammy/testing amd64 Packages
        100 /var/lib/dpkg/status
     8.0.35-27-1.jammy 500
        500 http://repo.percona.com/ps-80/apt jammy/testing amd64 Packages
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "ps-80",
				Component: "testing",
			},
			expectErr: nil,
		},
		{
			name:             "percona-postgresql-16_installed",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-postgresql-16:
Installed: 2:16.2-1.jammy
Candidate: 2:16.2-1.jammy
Version table:
*** 2:16.2-1.jammy 500
        500 http://repo.percona.com/ppg-16/apt jammy/main amd64 Packages
        100 /var/lib/dpkg/status
    2:16.0-1.jammy 500
        500 http://repo.percona.com/ppg-16/apt jammy/main amd64 Packages
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "ppg-16",
				Component: "release",
			},
			expectErr: nil,
		},
		{
			name:             "percona-release_installed",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-release:
Installed: 1.0-27.generic
Candidate: 1.0-27.generic
Version table:
*** 1.0-27.generic 500
        500 http://repo.percona.com/prel/apt jammy/main amd64 Packages
        100 /var/lib/dpkg/status
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "prel",
				Component: "release",
			},
			expectErr: nil,
		},
		{
			name:             "percona-toolkit_installed",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-toolkit:
Installed: 3.5.7-1.jammy
Candidate: 3.5.7-1.jammy
Version table:
*** 3.5.7-1.jammy 500
        500 http://repo.percona.com/percona/apt jammy/main amd64 Packages
        500 http://repo.percona.com/pt/apt jammy/main amd64 Packages
        500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
        100 /var/lib/dpkg/status
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "percona",
				Component: "release",
			},
			expectErr: nil,
		},
		{
			name:             "percona-xtrabackup-81_installed",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-xtrabackup-81:
Installed: 8.1.0-1-1.jammy
Candidate: 8.1.0-1-1.jammy
Version table:
*** 8.1.0-1-1.jammy 500
        500 http://repo.percona.com/percona/apt jammy/main amd64 Packages
        500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
        100 /var/lib/dpkg/status
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "percona",
				Component: "release",
			},
			expectErr: nil,
		},
		{
			name:             "etcd_installed",
			isPerconaPackage: isPerconaPackage("etcd"),
			repositoryOutput: []byte(`etcd:
Installed: 3.3.25+dfsg-7ubuntu0.22.04.1
Candidate: 3.3.25+dfsg-7ubuntu0.22.04.1
Version table:
*** 3.3.25+dfsg-7ubuntu0.22.04.1 500
		500 http://archive.ubuntu.com/ubuntu jammy-updates/universe amd64 Packages
		500 http://security.ubuntu.com/ubuntu jammy-security/universe amd64 Packages
		100 /var/lib/dpkg/status
	3.3.25+dfsg-7 500
		500 http://archive.ubuntu.com/ubuntu jammy/universe amd64 Packages
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "ubuntu",
				Component: "universe",
			},
			expectErr: nil,
		},
		{
			name:             "haproxy_installed",
			isPerconaPackage: isPerconaPackage("haproxy"),
			repositoryOutput: []byte(`haproxy:
Installed: 2.4.24-0ubuntu0.22.04.1
Candidate: 2.4.24-0ubuntu0.22.04.1
Version table:
*** 2.4.24-0ubuntu0.22.04.1 500
		500 http://archive.ubuntu.com/ubuntu jammy-updates/main amd64 Packages
		100 /var/lib/dpkg/status
	2.4.22-0ubuntu0.22.04.3 500
		500 http://security.ubuntu.com/ubuntu jammy-security/main amd64 Packages
	2.4.14-1ubuntu1 500
		500 http://archive.ubuntu.com/ubuntu jammy/main amd64 Packages
`),
			repositoryErr: nil,
			expectedRepository: &PackageRepository{
				Name:      "ubuntu",
				Component: "main",
			},
			expectErr: nil,
		},
		{
			name:               "unknown_package",
			isPerconaPackage:   isPerconaPackage("unknown"),
			repositoryOutput:   []byte(`N: Unable to locate package non_existing`),
			repositoryErr:      nil,
			expectedRepository: nil,
			expectErr:          errPackageRepositoryNotFound,
		},
		{
			name:             "package_not_installed",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-backup-mongodb:
Installed: 2.4.1-1.jammy
Candidate: 2.4.1-1.jammy
Version table:
	2.4.1-1.jammy 500
        500 http://repo.percona.com/pbm/apt jammy/main amd64 Packages
        500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
        100 /var/lib/dpkg/status
    2.4.0-1.jammy 500
        500 http://repo.percona.com/pbm/apt jammy/main amd64 Packages
        500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
`),
			repositoryErr:      nil,
			expectedRepository: nil,
			expectErr:          errPackageRepositoryNotFound,
		},
		{
			name:               "repository_error",
			isPerconaPackage:   isPerconaPackage("error"),
			repositoryOutput:   []byte(``),
			repositoryErr:      repositoryErr,
			expectedRepository: nil,
			expectErr:          repositoryErr,
		},
		{
			name:             "unexpected_repository_output",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-backup-mongodb:
Installed: 2.4.1-1.jammy
Candidate: 2.4.1-1.jammy
Version table:
*** 2.4.1-1.jammy 100
        100 /var/lib/dpkg/status
    2.4.0-1.jammy 500
        500 http://repo.percona.com/pbm/apt jammy/main amd64 Packages
        500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
`),
			repositoryErr:      nil,
			expectedRepository: nil,
			expectErr:          errUnexpectedRepoLine,
		},
		{
			name:             "no_candidate_repository_output",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-postgresql-16:
Installed: (none)
Candidate: (none)
Version table:
	2:16.2-1.jammy -1
		100 /var/lib/dpkg/status
`),
			repositoryErr:      nil,
			expectedRepository: nil,
			expectErr:          errPackageRepositoryNotFound,
		},
		{
			name:             "invalid_repository_output",
			isPerconaPackage: isPerconaPackage("percona-*"),
			repositoryOutput: []byte(`percona-backup-mongodb:
Installed: 2.4.1-1.jammy
Candidate: 2.4.1-1.jammy
Version table:
*** 2.4.1-1.jammy
        100 /var/lib/dpkg/status
    2.4.0-1.jammy 500
        500 http://repo.percona.com/pbm/apt jammy/main amd64 Packages
        500 http://repo.percona.com/tools/apt jammy/main amd64 Packages
`),
			repositoryErr:      nil,
			expectedRepository: nil,
			expectErr:          errUnexpectedConfiguredRepoLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, err := parseDebianRepositoryOutput(tt.repositoryOutput, tt.repositoryErr, tt.isPerconaPackage)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			}

			if tt.expectedRepository != nil {
				require.Equal(t, tt.expectedRepository, pkg)
			}
		})
	}
}
