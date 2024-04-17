package metrics

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

func queryRhelPackage(ctx context.Context, packageNamePattern string) ([]*Package, error) {
	args, err := getRhelPackageManager()
	if err != nil {
		return nil, err
	}
	args = append(args, "--qf", "'%{name}|%{version}|%{release}|%{from_repo}'", "--installed", packageNamePattern)
	zap.L().Sugar().Debugw("executing command", zap.String("cmd", strings.Join(args, " ")))

	cmdCtx, cancel := context.WithTimeout(ctx, pkgResultTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) // #nosec G204
	outputB, err := cmd.CombinedOutput()
	return parseRhelPackageOutput(outputB, err, isPerconaPackage(packageNamePattern))
}

func getRhelPackageManager() ([]string, error) {
	pkgMngs := [][]string{
		{"dnf", "repoquery"},
		{"yum", "repoquery"},
		{"repoquery"},
	}
	for _, pkgMng := range pkgMngs {
		if _, err := exec.LookPath(pkgMng[0]); err == nil {
			return pkgMng, nil
		}
	}
	return nil, errors.New("no package manager found")
}

func parseRhelPackageOutput(packageOutput []byte, rpmErr error, isPerconaPackage bool) ([]*Package, error) {
	if rpmErr != nil {
		// in case of package not found, rpm doesn't return error.
		// So if error is returned - something went wrong.
		zap.L().Sugar().Debugw("cmd output", zap.ByteString("output", packageOutput))
		return nil, rpmErr
	}

	scanner := bufio.NewScanner(bytes.NewReader(packageOutput))

	toReturn := make([]*Package, 0, 1)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " '\t")
		if len(line) == 0 {
			continue
		}

		tokens := strings.Split(line, "|")
		// The successful line for package shall be in format:
		// <package name>|<version>|<release>|<package repository>.
		// Example:
		// 'percona-xtrabackup-81|8.1.0|1.1.el8|tools-release-x86_64'
		// Note:
		// if package presents in 'packageOutput' it means it is installed,
		// no need to check package status.
		if len(tokens) != 4 {
			continue
		}

		pkgName, pkgVersion, pkgRelease, pkgRepository := tokens[0], tokens[1], tokens[2], tokens[3]
		toReturn = append(toReturn, &Package{
			Name:       pkgName,
			Version:    parseRhelPackageVersion(pkgVersion, pkgRelease, isPerconaPackage),
			Repository: parseRhelPackageRegistry(pkgRepository, isPerconaPackage),
		})
	}

	if err := scanner.Err(); err != nil {
		zap.L().Sugar().Warnw("failed to read output from rhel package manager", zap.Error(err))
		return nil, err
	}

	if len(toReturn) == 0 {
		// no installed packaged found matching pkgNamePattern
		return nil, errPackageNotFound
	}
	return toReturn, nil
}

func parseRhelPackageVersion(packageVersion, packageRelease string, isPerconaPackage bool) string {
	// Rhel package has a separate fields for version and release values:
	// Example:
	// version = '2.5', '8.1.0'
	// release = '1.el8', '3.2.el9'

	// need to trim extra distribution name from the end.
	// Distribution name may be at the end of:
	// - packageRelease
	// or
	// - packageVersion, if packageRelease is empty.
	if len(packageRelease) != 0 {
		if pos := strings.LastIndex(packageRelease, "."); pos != -1 {
			packageRelease = packageRelease[0:pos]
		}
	} else if pos := strings.LastIndex(packageVersion, "."); pos != -1 {
		packageVersion = packageVersion[0:pos]
	}

	if isPerconaPackage && len(packageRelease) != 0 {
		packageRelease = strings.ReplaceAll(packageRelease, ".", "-")
		// need to join them with '-' separator.
		return fmt.Sprintf("%s-%s", packageVersion, packageRelease)
	}
	return packageVersion
}

func parseRhelPackageRegistry(packageRepository string, isPerconaPackage bool) PackageRepository {
	// packageRepository contains info about package repository name where package comes from.
	// Example:
	// packageRepository = 'pt-release-x86_64', 'noarch', ''
	// Note: repository value may be empty!

	var toReturn PackageRepository
	if len(packageRepository) == 0 {
		return toReturn
	}

	if !isPerconaPackage {
		toReturn.Name = packageRepository
		return toReturn
	}

	// need to trim extra arch (-x86_64) from the end.
	if pos := strings.LastIndex(packageRepository, "-"); pos != -1 {
		packageRepository = packageRepository[0:pos]
	}

	// Percona repository name has format:
	// <name>-<component>
	// Example:
	// 'ps-80-release'
	// where 'ps-80' is name and 'release' is component.
	// need to split them.
	if pos := strings.LastIndex(packageRepository, "-"); pos != -1 {
		toReturn.Name = packageRepository[0:pos]
		toReturn.Component = packageRepository[pos+1:]
	}
	return toReturn
}

// getRhelExternalPackages returns list of external package patterns that are unique for RHEL systems.
func getRhelExternalPackages() []string {
	return []string{
		// PG extensions
		"wal2json*",
	}
}
