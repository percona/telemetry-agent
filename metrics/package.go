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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	debVersion "github.com/knqyf263/go-deb-version"
	"go.uber.org/zap"
)

const (
	pkgResultTimeout = 30 * time.Second
)

var errPackageNotFound = errors.New("package is not found")

// NOTE: the logic in this file is designed in a way "do our best to provide value", i.e. in case an error appears
// it is not passed to upper level but is just printed into log stream and fallback value is applied.

// Package represents a software package with its name and version.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// queryPkgFunc represents a function type for querying package information from particular package manager (dpkg or rpm).
type queryPkgFunc func(ctx context.Context, packageName string) ([]*Package, error)

// ScrapeInstalledPackages scrapes the installed packages on the host and returns a slice of Package structs along with any errors encountered.
// The function uses the localOs variable to determine the package manager to use.
func ScrapeInstalledPackages(ctx context.Context) []*Package {
	pkgList := getCommonPerconaPackages()
	pkgList = append(pkgList, getCommonExternalPackages()...)
	localOs := getOSInfo()

	toReturn := make([]*Package, 0, 1)
	var pkgFunc queryPkgFunc

	switch {
	case isDebianFamily(localOs):
		pkgFunc = queryDpkg
	case isRHELFamily(localOs):
		pkgFunc = queryRpm
	default:
		zap.L().Sugar().Warnw("unsupported package system", zap.String("OS", localOs))
		return toReturn
	}

	for _, pkgNamePattern := range pkgList {
		pkgL, err := pkgFunc(ctx, pkgNamePattern)
		if err != nil {
			if !errors.Is(err, errPackageNotFound) {
				zap.L().Sugar().Warnw("failed to get package info", zap.Error(err), zap.String("package", pkgNamePattern))
			}
			// go to next package pattern silently
			continue
		}
		// packages are installed
		toReturn = append(toReturn, pkgL...)
	}
	return toReturn
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

func queryDpkg(ctx context.Context, packageNamePattern string) ([]*Package, error) {
	args := []string{"dpkg-query", "-f", "'${db:Status-Abbrev}|${binary:Package}|${source:Version}\n'", "-W", packageNamePattern}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseDpkgOutput(packageNamePattern, outputB, err)
}

func parseDpkgOutput(packageNamePattern string, dpkgOutput []byte, dpkgErr error) ([]*Package, error) { //nolint:cyclop
	if dpkgErr != nil {
		if strings.Contains(string(dpkgOutput), "no packages found matching") {
			// package is not installed
			return nil, errPackageNotFound
		}

		zap.L().Sugar().Debugw("cmd output", zap.ByteString("output", dpkgOutput))
		return nil, dpkgErr
	}

	scanner := bufio.NewScanner(bytes.NewReader(dpkgOutput))
	toReturn := make([]*Package, 0, 1)
	for scanner.Scan() {
		// trim spaces and single quote chars
		line := strings.Trim(scanner.Text(), " '")
		if len(line) == 0 {
			continue
		}

		tokens := strings.Split(line, "|")
		// The successful line for package shall be in format:
		// <status> |<package name>|[epoch:]<version>.
		// Example:
		// 'ii |percona-xtrabackup-81|8.1.0-1-1.jammy'
		// or with epoch:
		// 'ii |percona-xtrabackup-81|2:8.1.0-1-1.jammy'
		if len(tokens) != 3 {
			continue
		}

		pkgStatus, pkgName, pkgVersion := tokens[0], tokens[1], tokens[2]

		// check package status first.
		pkgStatus = strings.TrimSpace(pkgStatus)
		if pkgStatus != "ii" && pkgStatus != "iHR" {
			// package is not installed, skip it.
			continue
		}

		// process package name
		pkgName = processDebianPackageName(pkgName)
		if len(pkgName) == 0 {
			continue
		}

		// process package version
		pkgVersion = processDebianPackageVersion(packageNamePattern, pkgVersion)
		if len(pkgVersion) == 0 {
			continue
		}

		toReturn = append(toReturn, &Package{
			Name:    pkgName,
			Version: pkgVersion,
		})
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("failed to read output from dpkg-query", zap.Error(err))
		return nil, err
	}

	if len(toReturn) == 0 {
		// no installed packaged found matching pkgNamePattern
		return nil, errPackageNotFound
	}
	return toReturn, nil
}

func processDebianPackageName(pkgName string) string {
	pkgName = strings.TrimSpace(pkgName)
	// pkgName may have format:
	// <name>[:architecture]
	// Example:
	// 'percona-xtrabackup-81:amd64'
	// Need to trim architecture part.
	return strings.Split(pkgName, ":")[0]
}

func isPerconaPackage(packageNamePattern string) bool {
	for _, pkgPattern := range getCommonPerconaPackages() {
		if packageNamePattern == pkgPattern {
			return true
		}
	}
	return false
}

func processDebianPackageVersion(packageNamePattern, pkgVersion string) string {
	// Debian package version have format:
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	// [epoch:]upstream_version[-debian_revision]
	// Example:
	// upstream_version = '8.1.0'
	// upstream_version-debian_revision = '8.1.0-1.1', '7.81.0-1ubuntu1.16'
	// epoch:upstream_version-debian_revision = '2:8.1.0-1.1', '1:7.81.0-1ubuntu1.16'
	//
	// But Percona packages have differences in [-debian_revision] part:
	// upstream_version-debian_revision = '8.2.0-1-1.jammy'
	// here '.jammy' is distribution name.

	if isPerconaPackage(packageNamePattern) {
		// Percona's package version case.
		// need to trim distribution name from the end.
		if pos := strings.LastIndex(pkgVersion, "."); pos != -1 {
			pkgVersion = pkgVersion[0:pos]
		}

		v, err := debVersion.NewVersion(pkgVersion)
		if err != nil {
			return pkgVersion
		}

		if len(v.Revision()) != 0 {
			// special hack - replace all "." with "-" to unify version format
			// for all Percona's packages.
			revision := strings.ReplaceAll(v.Revision(), ".", "-")
			return fmt.Sprintf("%s-%s", v.Version(), revision)
		}
		return v.Version()
	}

	// Regular Debian package case.
	v, err := debVersion.NewVersion(pkgVersion)
	if err != nil {
		return pkgVersion
	}
	pkgVersion = v.Version()
	// need to trim '+dfsg' part if it is present.
	if pos := strings.Index(pkgVersion, "+dfsg"); pos != -1 {
		pkgVersion = pkgVersion[0:pos]
	}
	return pkgVersion
}

func queryRpm(ctx context.Context, packageNamePattern string) ([]*Package, error) {
	args := []string{"rpm", "-a", "-q", packageNamePattern, "--queryformat", "'%{NAME}|%{VERSION}|%{RELEASE}\n'"}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseRpmOutput(packageNamePattern, outputB, err)
}

func parseRpmOutput(packageNamePattern string, rpmOutput []byte, rpmErr error) ([]*Package, error) {
	if rpmErr != nil {
		// in case of package not found, rpm doesn't return error.
		// So if error is returned - something went wrong.
		zap.L().Sugar().Debugw("cmd output", zap.ByteString("output", rpmOutput))
		return nil, rpmErr
	}

	scanner := bufio.NewScanner(bytes.NewReader(rpmOutput))

	toReturn := make([]*Package, 0, 1)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " '")
		if len(line) == 0 {
			continue
		}

		tokens := strings.Split(line, "|")
		// The successful line for package shall be in format:
		// <package name>|<version>|<release>.
		// Example:
		// 'percona-xtrabackup-81|8.1.0|1.1.el8'
		// Note:
		// if package presents in 'rpmOutput' it means it is installed,
		// no need to check package status.
		if len(tokens) != 3 {
			continue
		}

		pkgName, pkgVersion, pkgRelease := tokens[0], tokens[1], tokens[2]
		version := processRhelPackageVersion(packageNamePattern, pkgVersion, pkgRelease)

		if len(version) > 0 {
			toReturn = append(toReturn, &Package{
				Name:    pkgName,
				Version: version,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("failed to read output from rpm", zap.Error(err))
		return nil, err
	}

	if len(toReturn) == 0 {
		// no installed packaged found matching pkgNamePattern
		return nil, errPackageNotFound
	}
	return toReturn, nil
}

func processRhelPackageVersion(packageNamePattern, pkgVersion, pkgRelease string) string {
	// Rhel package has a separate fields for version and release values:
	// Example:
	// version = '2.5', '8.1.0'
	// release = '1.el8', '3.2.el9'

	// need to trim extra distribution name from the end.
	// Distribution name may be at the end of:
	// - pkgRelease
	// or
	// - pkgVersion, if pkgRelease is empty.
	if len(pkgRelease) != 0 {
		if pos := strings.LastIndex(pkgRelease, "."); pos != -1 {
			pkgRelease = pkgRelease[0:pos]
		}
	} else if pos := strings.LastIndex(pkgVersion, "."); pos != -1 {
		pkgVersion = pkgVersion[0:pos]
	}

	if isPerconaPackage(packageNamePattern) && len(pkgRelease) != 0 {
		pkgRelease = strings.ReplaceAll(pkgRelease, ".", "-")
		// need to join them with '-' separator.
		return fmt.Sprintf("%s-%s", pkgVersion, pkgRelease)
	}
	return pkgVersion
}

// getCommonPerconaPackages returns list of Percona package patterns that have the same names both on Debian and RHEL systems.
func getCommonPerconaPackages() []string {
	return []string{
		"Percona-*",
		"percona-*",
		"proxysql*",
		"pmm*",
	}
}

// getCommonExternalPackages returns list of non Percona package patterns that have the same names both on Debian and RHEL systems.
func getCommonExternalPackages() []string {
	return []string{
		// PG extensions
		"etcd*",
		"haproxy",
		"patroni",
		"pg*",
		"postgis",
		"wal2json",
	}
}
