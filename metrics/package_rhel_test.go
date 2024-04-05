package metrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsRHELFamily(t *testing.T) {
	t.Parallel()

	for _, tt := range osNames {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, getDistroFamily(tt.osName))
		})
	}
}

func TestParseRhelPackageOutput(t *testing.T) {
	t.Parallel()

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
			packageOutput: []byte(`percona-server-server|8.0.36|28.1.el9|ps-80-release-x86_64
percona-mysql-shell|8.0.36|1.el9|ps-80-release-x86_64
percona-mongodb-mongosh|2.1.1|1.el9|pdmdb-7.0-release-x86_64
percona-server-mongodb-server|7.0.5|3.el9|pdmdb-7.0-release-x86_64
percona-postgresql16|16.2|2.el9|ppg-16-release-x86_64
percona-postgresql16-server|16.2|2.el9|ppg-16-release-x86_64
percona-pg_stat_monitor16|2.0.4|2.el9|ppg-16-release-x86_64
percona-pgaudit16|16.0|2.el9|ppg-16-release-x86_64
percona-wal2json16|2.5|2.el9|ppg-16-release-x86_64
percona-pgbouncer|1.22.0|1.el9|ppg-16-release-x86_64
percona-server-mongodb|7.0.5|3.el9|pdmdb-7.0-release-x86_64
percona-xtrabackup-81|8.1.0|1.1.el9|tools-release-x86_64
percona-toolkit|3.5.7|1.el9|pt-release-x86_64
percona-backup-mongodb|2.4.1.el9||pbm-release-x86_64
`),
			packageErr: nil,
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
					Name:    "percona-mysql-shell",
					Version: "8.0.36-1",
					Repository: PackageRepository{
						Name:      "ps-80",
						Component: "release",
					},
				},
				{
					Name:    "percona-mongodb-mongosh",
					Version: "2.1.1-1",
					Repository: PackageRepository{
						Name:      "pdmdb-7.0",
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
					Name:    "percona-postgresql16",
					Version: "16.2-2",
					Repository: PackageRepository{
						Name:      "ppg-16",
						Component: "release",
					},
				},
				{
					Name:    "percona-postgresql16-server",
					Version: "16.2-2",
					Repository: PackageRepository{
						Name:      "ppg-16",
						Component: "release",
					},
				},
				{
					Name:    "percona-pg_stat_monitor16",
					Version: "2.0.4-2",
					Repository: PackageRepository{
						Name:      "ppg-16",
						Component: "release",
					},
				},
				{
					Name:    "percona-pgaudit16",
					Version: "16.0-2",
					Repository: PackageRepository{
						Name:      "ppg-16",
						Component: "release",
					},
				},
				{
					Name:    "percona-wal2json16",
					Version: "2.5-2",
					Repository: PackageRepository{
						Name:      "ppg-16",
						Component: "release",
					},
				},
				{
					Name:    "percona-pgbouncer",
					Version: "1.22.0-1",
					Repository: PackageRepository{
						Name:      "ppg-16",
						Component: "release",
					},
				},
				{
					Name:    "percona-server-mongodb",
					Version: "7.0.5-3",
					Repository: PackageRepository{
						Name:      "pdmdb-7.0",
						Component: "release",
					},
				},
				{
					Name:    "percona-xtrabackup-81",
					Version: "8.1.0-1-1",
					Repository: PackageRepository{
						Name:      "tools",
						Component: "release",
					},
				},
				{
					Name:    "percona-toolkit",
					Version: "3.5.7-1",
					Repository: PackageRepository{
						Name:      "pt",
						Component: "release",
					},
				},
				{
					Name:    "percona-backup-mongodb",
					Version: "2.4.1",
					Repository: PackageRepository{
						Name:      "pbm",
						Component: "release",
					},
				},
			},
			expectErr: nil,
		},
		{
			name:             "pattern_percona_proxysql_installed",
			isPerconaPackage: isPerconaPackage("proxysql*"),
			packageOutput: []byte(`proxysql|1.5.5|1.2.el9|proxysql-release-x86_64
proxysql2|2.5.5|1.2.el9|proxysql-release-x86_64`),
			packageErr: nil,
			expectedPackageList: []*Package{
				{
					Name:    "proxysql",
					Version: "1.5.5-1-2",
					Repository: PackageRepository{
						Name:      "proxysql",
						Component: "release",
					},
				},
				{
					Name:    "proxysql2",
					Version: "2.5.5-1-2",
					Repository: PackageRepository{
						Name:      "proxysql",
						Component: "release",
					},
				},
			},
			expectErr: nil,
		},
		{
			name:             "pattern_percona_pmm_installed",
			isPerconaPackage: isPerconaPackage("pmm*"),
			packageOutput:    []byte(`pmm2-client|2.41.2|6.1.el9|pmm2-client-testing-x86_64`),
			packageErr:       nil,
			expectedPackageList: []*Package{
				{
					Name:    "pmm2-client",
					Version: "2.41.2-6-1",
					Repository: PackageRepository{
						Name:      "pmm2-client",
						Component: "testing",
					},
				},
			},
			expectErr: nil,
		},
		{
			name:             "exact_external_installed",
			isPerconaPackage: isPerconaPackage("etcd"),
			packageOutput:    []byte(`etcd|3.5.12|1.el8|ppg-16-release-x86_64`),
			packageErr:       nil,
			expectedPackageList: []*Package{
				{
					Name:    "etcd",
					Version: "3.5.12",
					Repository: PackageRepository{
						Name:      "ppg-16-release-x86_64",
						Component: "",
					},
				},
			},
			expectErr: nil,
		},
		{
			name:                "percona_not_installed",
			isPerconaPackage:    isPerconaPackage("percona-*"),
			packageOutput:       []byte(``),
			packageErr:          nil,
			expectedPackageList: nil,
			expectErr:           errPackageNotFound,
		},
		{
			name:                "external_not_installed",
			isPerconaPackage:    isPerconaPackage("etcd"),
			packageOutput:       []byte(``),
			packageErr:          nil,
			expectedPackageList: nil,
			expectErr:           errPackageNotFound,
		},
		{
			name:                "rpm_error",
			isPerconaPackage:    isPerconaPackage("percona-*"),
			packageOutput:       []byte(``),
			packageErr:          errors.New("rpm: --test may only be specified during package installation and erasure"),
			expectedPackageList: nil,
			expectErr:           errors.New("rpm: --test may only be specified during package installation and erasure"),
		},
		{
			name:                "invalid_rpm_output",
			isPerconaPackage:    isPerconaPackage("percona-xtrabackup-81"),
			packageOutput:       []byte(`percona-xtrabackup-81|`),
			packageErr:          nil,
			expectedPackageList: nil,
			expectErr:           errPackageNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, err := parseRhelPackageOutput(tt.packageOutput, tt.packageErr, tt.isPerconaPackage)
			if tt.expectErr != nil {
				require.ErrorAs(t, err, &tt.expectErr)
			}

			if tt.expectedPackageList != nil {
				require.Equal(t, tt.expectedPackageList, pkg)
			}
		})
	}
}
