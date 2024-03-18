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
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	pkgResultTimeout = 30 * time.Second
)

var errPackageNotFound = errors.New("package is not found")

// Package represents a software package with its name and version.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// queryPkgFunc represents a function type for querying package information from particular package manager (dpkg or rpm).
type queryPkgFunc func(ctx context.Context, packageName string) (*Package, error)

// ScrapeInstalledPackages scrapes the installed packages on the host and returns a slice of Package structs along with any errors encountered.
// The function uses the localOs variable to determine the package manager to use.
func ScrapeInstalledPackages(ctx context.Context) ([]*Package, error) {
	var pkgFunc queryPkgFunc
	pkgList := getCommonPackages()
	localOs := getOSInfo()

	switch {
	case isDebianFamily(localOs):
		pkgFunc = queryDpkg
		pkgList = append(pkgList, getDebianPackages()...)
	case isRHELFamily(localOs):
		pkgFunc = queryRpm
		pkgList = append(pkgList, getRhelPackages()...)
	default:
		zap.L().Sugar().Warnw("unsupported package system", zap.String("OS", localOs))
		return nil, fmt.Errorf("unsupported package system")
	}

	toReturn := make([]*Package, 0, len(pkgList))
	var pkg *Package
	var err error
	for _, pName := range pkgList {
		if pkg, err = pkgFunc(ctx, pName); err != nil {
			if !errors.Is(err, errPackageNotFound) {
				zap.L().Sugar().Warnw("can't get package info", zap.Error(err), zap.String("package", pName))
			}
			// go to next package silently
			continue
		}
		// package is installed
		toReturn = append(toReturn, pkg)
	}
	return toReturn, nil
}

func isDebianFamily(name string) bool {
	nameL := strings.ToLower(name)
	prefixes := []string{"debian", "ubuntu"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(nameL, prefix) {
			return true
		}
	}
	return false
}

func isRHELFamily(name string) bool {
	nameL := strings.ToLower(name)
	prefixes := []string{"el", "centos", "oracle", "rocky", "red hat", "amazon", "alma"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(nameL, prefix) {
			return true
		}
	}
	return false
}

func queryDpkg(ctx context.Context, packageName string) (*Package, error) {
	args := []string{"dpkg-query", "-f", "'${Package} ${db:Status-Abbrev}${Version}'", "-W", packageName}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseDpkgOutput(packageName, string(outputB), err)
}

func parseDpkgOutput(packageName, dpkgOutput string, dpkgErr error) (*Package, error) { //nolint:cyclop
	if dpkgErr != nil {
		if strings.Contains(dpkgOutput, "no packages found matching") {
			// package is not installed
			return nil, errPackageNotFound
		}

		zap.L().Sugar().Debugw("cmd output", zap.String("output", dpkgOutput))
		return nil, fmt.Errorf("error listing installed packages: %w", dpkgErr)
	}

	scanner := bufio.NewScanner(strings.NewReader(dpkgOutput))

	var version string
	for scanner.Scan() {
		// trim spaces and single quote chars
		line := strings.Trim(scanner.Text(), " '")
		if len(line) == 0 {
			continue
		}

		tokens := strings.Split(line, " ")
		if tokens[0] != packageName {
			continue
		}

		if len(tokens) == 2 {
			// the output is:
			// '<package name> <state>'
			// package is known but state != Installed
			continue
		}
		// The successful line for package shall be in format:
		// <package name> <status> <version>.
		// Example:
		// 'percona-xtrabackup-81 ii 8.1.0-1-1.jammy'
		if len(tokens) != 3 {
			continue
		}

		if tokens[1] == "ii" {
			version = tokens[2]
			// need to trim extra chars from release part
			if pos := strings.LastIndex(version, "."); pos != -1 {
				version = version[0:pos]
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error listing installed packages: %w", err)
	}

	if len(version) > 0 {
		return &Package{
			Name:    packageName,
			Version: version,
		}, nil
	}

	// no installed packaged found
	return nil, errPackageNotFound
}

func queryRpm(ctx context.Context, packageName string) (*Package, error) {
	args := []string{"rpm", "-q", packageName, "--queryformat", "'%{NAME} %{VERSION} %{RELEASE}'"}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseRpmOutput(packageName, string(outputB), err)
}

func parseRpmOutput(packageName, rpmOutput string, rpmErr error) (*Package, error) {
	if rpmErr != nil {
		if strings.Contains(rpmOutput, "is not installed") {
			// package is not installed
			return nil, errPackageNotFound
		}

		zap.L().Sugar().Debugw("cmd output", zap.String("output", rpmOutput))
		return nil, fmt.Errorf("error listing installed packages: %w", rpmErr)
	}

	scanner := bufio.NewScanner(strings.NewReader(rpmOutput))

	var version string
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " '")
		if len(line) == 0 {
			continue
		}

		tokens := strings.Split(line, " ")
		if len(tokens) != 3 {
			return nil, fmt.Errorf("error parsing rpm line %q", line)
		}

		if tokens[0] != packageName {
			return nil, fmt.Errorf("error parsing rpm line %q", line)
		}
		release := tokens[2]
		// need to trim extra chars from release part
		if pos := strings.LastIndex(release, "."); pos != -1 {
			release = release[0:pos]
		}

		release = strings.ReplaceAll(release, ".", "-")

		version = strings.Join([]string{tokens[1], release}, "-")
		break
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error listing installed packages, scanner error: %w", err)
	}

	if len(version) > 0 {
		return &Package{
			Name:    packageName,
			Version: version,
		}, nil
	}
	// package is not installed
	return nil, errPackageNotFound
}

// getDebianPackages returns list of Percona's Debian specific package names.
func getDebianPackages() []string {
	return []string{
		// PS + PXC packages
		"Percona-Server-server-5.7",         // deb
		"Percona-Xtradb-Cluster-server-5.7", // deb
		// PG
		"percona-postgresql-14", // deb
		"percona-postgresql-15", // deb
		"percona-postgresql-16", // deb
	}
}

// getRhelPackages returns list of Percona's RHEL specific package names.
func getRhelPackages() []string {
	return []string{
		// PS + PXC packages
		"Percona-Server-server-57",         // rpm
		"Percona-XtraDB-Cluster-server-57", // rpm
		// PG
		"percona-postgresql14-server", // rpm
		"percona-postgresql15-server", // rpm
		"percona-postgresql16-server", // rpm
	}
}

// getCommonPackages returns list of Percona packages that have the same names both on Debian and RHEL systems.
func getCommonPackages() []string {
	return []string{
		// PS + PXC packages
		"percona-server-server",
		"percona-server-server-pro",
		"percona-mysql-shell",
		"percona-mysql-router",
		"percona-mysql-router-pro",
		"proxysql",
		"proxysql2",
		"percona-orchestrator",
		"percona-xtradb-cluster-server",
		"percona-xtradb-cluster-mysql-router",
		// PSMDB packages
		"percona-server-mongodb-server",
		"percona-server-mongodb-server-pro",
		"percona-server-mongodb-mongos",
		"percona-server-mongodb-mongos-pro",
		// PBM
		"percona-backup-mongodb",
		// PG
		"etcd",
		"percona-pgbouncer",
		// PXB
		"percona-xtrabackup-24",
		"percona-xtrabackup-80",
		"percona-xtrabackup-81",
		"percona-xtrabackup-82",
		"percona-xtrabackup-83",
		// Percona Toolkit
		"percona-toolkit",
		// HA proxy
		"percona-haproxy",
		// PMM Agent
		"pmm2-client",
	}
}
