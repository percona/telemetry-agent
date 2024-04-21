package metrics

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	debVersion "github.com/knqyf263/go-deb-version"
	"go.uber.org/zap"
)

var (
	errUnexpectedRepoLine           = errors.New("unexpected package repository line")
	errUnexpectedConfiguredRepoLine = errors.New("unexpected configured package repository line")
)

func queryDebianPackage(ctx context.Context, packageNamePattern string) ([]*Package, error) {
	args := []string{"dpkg-query", "-f", "'${db:Status-Abbrev}|${binary:Package}|${source:Version}\n'", "-W", packageNamePattern}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseDebianPackageOutput(outputB, err, isPerconaPackage(packageNamePattern))
}

func parseDebianPackageOutput(dpkgOutput []byte, dpkgErr error, isPerconaPackage bool) ([]*Package, error) { //nolint:cyclop
	if dpkgErr != nil {
		if strings.Contains(dpkgErr.Error(), "no packages found matching") {
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
		pkgName = parseDebianPackageName(pkgName)
		if len(pkgName) == 0 {
			continue
		}

		// process package version
		pkgVersion = parseDebianPackageVersion(pkgVersion, isPerconaPackage)
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

func parseDebianPackageName(pkgName string) string {
	pkgName = strings.TrimSpace(pkgName)
	// pkgName may have format:
	// <name>[:architecture]
	// Example:
	// 'percona-xtrabackup-81:amd64'
	// Need to trim architecture part.
	return strings.Split(pkgName, ":")[0]
}

func parseDebianPackageVersion(pkgVersion string, isPerconaPackage bool) string {
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

	if isPerconaPackage {
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

func queryDebianRepository(ctx context.Context, packageName string, isPerconaPackage bool) (*PackageRepository, error) {
	args := []string{"apt-cache", "-q=0", "policy", packageName}
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseDebianRepositoryOutput(outputB, err, isPerconaPackage)
}

func parseDebianRepositoryOutput(repoOutput []byte, repoErr error, isPerconaPackage bool) (*PackageRepository, error) { //nolint:cyclop
	if repoErr != nil {
		zap.L().Sugar().Debugw("cmd output", zap.ByteString("output", repoOutput))
		return nil, repoErr
	}

	// the output example:
	// percona-server-server:
	//  Installed: 8.0.36-28-1.jammy
	//  Candidate: 8.0.36-28-1.jammy
	//  Version table:
	// *** 8.0.36-28-1.jammy 500
	//        500 http://repo.percona.com/ps-80/apt jammy/main amd64 Packages
	//        100 /var/lib/dpkg/status
	//     8.0.35-27-1.jammy 500
	//        500 http://repo.percona.com/ps-80/apt jammy/main amd64 Packages
	scanner := bufio.NewScanner(bytes.NewReader(repoOutput))
	for scanner.Scan() {
		// trim spaces and single quote chars
		line := strings.Trim(scanner.Text(), " '\t")
		if strings.Contains(line, "Unable to locate package") {
			// package is not installed
			return nil, errPackageRepositoryNotFound
		}
		// need to find line that refers to installed package
		if len(line) == 0 || !strings.HasPrefix(line, "***") {
			continue
		}

		// line has format:
		// *** <version>.<distribution> <priority>
		// need to find this <priority> in the following lines
		tokens := strings.Split(line, " ")
		if len(tokens) != 3 {
			// smth strange
			zap.L().Sugar().Warnw("unexpected configured package repository line", zap.String("line", line))
			return nil, errUnexpectedConfiguredRepoLine
		}
		priority := tokens[2]
		for scanner.Scan() {
			line = strings.Trim(scanner.Text(), " '\t")
			if len(line) == 0 || !strings.HasPrefix(line, priority) {
				continue
			}
			// found needed repository line, parse it
			return parseDebianPackageRepositoryLine(line, isPerconaPackage)
		}
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("failed to read output from apt-cache", zap.Error(err))
		return nil, err
	}

	// no package repository found
	return nil, errPackageRepositoryNotFound
}

func parseDebianPackageRepositoryLine(repositoryLine string, isPerconaPackage bool) (*PackageRepository, error) {
	// repository line has format:
	// <priority> <url> <distribution>/<repository_branch> <arch> ....
	// or
	// <priority> <filesystem path>
	repoTokens := strings.Split(repositoryLine, " ")
	if len(repoTokens) < 3 {
		// this is case with filesystem path or smth strange
		zap.L().Sugar().Warnw("unexpected package repository line", zap.String("line", repositoryLine))
		return nil, errUnexpectedRepoLine
	}

	repoAddr := repoTokens[1]
	repoURL, err := url.Parse(repoAddr)
	if err != nil {
		zap.L().Sugar().Warnw("failed to parse repository url", zap.Error(err), zap.String("url", repoAddr))
		return nil, err
	}
	repoName := strings.Split(strings.Trim(repoURL.Path, "/"), "/")[0]

	var repoComponent string
	if repoBranch := strings.Split(repoTokens[2], "/"); len(repoBranch) == 2 {
		repoComponent = repoBranch[1]
	}
	if isPerconaPackage && repoComponent == "main" {
		repoComponent = "release"
	}
	return &PackageRepository{
		Name:      repoName,
		Component: repoComponent,
	}, nil
}

// getDebianPerconaPackages returns list of Percona package patterns that are unique for Debian systems.
func getDebianPerconaPackages() []string {
	return []string{
		"Percona-*",
	}
}

// getDebianExternalPackages returns list of external package patterns that are unique for Debian systems.
func getDebianExternalPackages() []string {
	return []string{
		// PG extensions
		"postgresql-*",
	}
}
