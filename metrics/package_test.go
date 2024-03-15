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
	"fmt"
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

func TestIsDebianFamily(t *testing.T) { //nolint:tparallel
	t.Parallel()

	for _, tt := range osNames { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedDebian, isDebianFamily(tt.osName))
		})
	}
}

func TestIsRHELFamily(t *testing.T) { //nolint:tparallel
	t.Parallel()

	for _, tt := range osNames { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedRhel, isRHELFamily(tt.osName))
		})
	}
}

func TestParseDpkgOutput(t *testing.T) { //nolint:tparallel
	t.Parallel()

	tests := []struct {
		name        string
		packageName string
		dpkgOutput  string
		dpkgErr     error
		expectedPkg *Package
		expectErr   error
	}{
		{
			name:        "PackageInstalled",
			packageName: "percona-xtrabackup-81",
			dpkgOutput:  "percona-xtrabackup-81 ii 8.1.0-1-1.jammy",
			dpkgErr:     nil,
			expectedPkg: &Package{
				Name:    "percona-xtrabackup-81",
				Version: "8.1.0-1-1",
			},
			expectErr: nil,
		},
		{
			name:        "PackageNotInstalled",
			packageName: "percona-xtrabackup-81",
			dpkgOutput:  "no packages found matching percona-xtrabackup-81",
			dpkgErr:     errors.New("no packages found matching percona-xtrabackup-81"),
			expectedPkg: nil,
			expectErr:   errPackageNotFound,
		},
		{
			name:        "DpkgError",
			packageName: "percona-xtrabackup-81",
			dpkgOutput:  "",
			dpkgErr:     errors.New("dpkg-query: error while loading shared libraries: libapt-pkg.so.6.0: cannot open shared object file: No such file or directory"),
			expectedPkg: nil,
			expectErr:   fmt.Errorf("error listing installed packages: %w", errors.New("dpkg-query: error while loading shared libraries: libapt-pkg.so.6.0: cannot open shared object file: No such file or directory")),
		},
		{
			name:        "InvalidDpkgOutput",
			packageName: "percona-xtrabackup-81",
			dpkgOutput:  "percona-xtrabackup-81 ii",
			dpkgErr:     nil,
			expectedPkg: nil,
			expectErr:   errPackageNotFound,
		},
		{
			name:        "PackageUnknown",
			packageName: "percona-xtrabackup-81",
			dpkgOutput:  "percona-xtrabackup-81 un",
			dpkgErr:     nil,
			expectedPkg: nil,
			expectErr:   errPackageNotFound,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := parseDpkgOutput(tt.packageName, tt.dpkgOutput, tt.dpkgErr)
			if tt.expectErr != nil {
				require.ErrorAs(t, err, &tt.expectErr)
			}

			if tt.expectedPkg != nil {
				require.Equal(t, tt.expectedPkg, pkg)
			}
		})
	}
}

func TestParseRpmOutput(t *testing.T) { //nolint:tparallel
	t.Parallel()

	tests := []struct {
		name        string
		packageName string
		rpmOutput   string
		rpmErr      error
		expectedPkg *Package
		expectErr   error
	}{
		{
			name:        "PackageInstalled",
			packageName: "percona-xtradb-cluster-server",
			rpmOutput:   "percona-xtradb-cluster-server 8.0.35 27.1.el8",
			rpmErr:      nil,
			expectedPkg: &Package{
				Name:    "percona-xtradb-cluster-server",
				Version: "8.0.35-27-1",
			},
			expectErr: nil,
		},
		{
			name:        "PackageNotInstalled",
			packageName: "percona-xtradb-cluster-server",
			rpmOutput:   "package percona-xtradb-cluster-server is not installed",
			rpmErr:      errors.New("package percona-xtradb-cluster-server is not installed"),
			expectedPkg: nil,
			expectErr:   errPackageNotFound,
		},
		{
			name:        "RpmError",
			packageName: "percona-xtradb-cluster-server",
			rpmOutput:   "",
			rpmErr:      errors.New("rpm: -x: unknown option"),
			expectedPkg: nil,
			expectErr:   fmt.Errorf("error listing installed packages: %w", errors.New("rpm: -x: unknown option")),
		},
		{
			name:        "InvalidRpmOutput",
			packageName: "percona-xtradb-cluster-server",
			rpmOutput:   "'percona-xtradb-cluster-server 8.0.35'",
			rpmErr:      nil,
			expectedPkg: nil,
			expectErr:   fmt.Errorf("error parsing rpm line %q", "'percona-xtradb-cluster-server 8.0.35'"),
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := parseRpmOutput(tt.packageName, tt.rpmOutput, tt.rpmErr)
			if tt.expectErr != nil {
				require.ErrorAs(t, err, &tt.expectErr)
			}

			if tt.expectedPkg != nil {
				require.Equal(t, tt.expectedPkg, pkg)
			}
		})
	}
}
