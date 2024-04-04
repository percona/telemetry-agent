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
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

const (
	pkgResultTimeout = 30 * time.Second
)

var (
	errPackageNotFound           = errors.New("package is not found")
	errPackageRepositoryNotFound = errors.New("package repository is not found")
)

// NOTE: the logic in this file is designed in a way "do our best to provide value", i.e. in case an error appears
// it is not passed to upper level but is just printed into log stream and fallback value is applied.

// PackageRepository represents a repository where a software package is located.
type PackageRepository struct {
	Name      string `json:"name"`
	Component string `json:"component"`
}

// Package represents a software package with its name and version.
type Package struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Repository PackageRepository `json:"repository"`
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
		pkgFunc = queryDebianPackage
		pkgList = append(pkgList, getDebianPerconaPackages()...)
	case isRHELFamily(localOs):
		pkgFunc = queryRhelPackage
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
		if isDebianFamily(localOs) {
			// need extra processing - get package repository info.
			for _, pkg := range pkgL {
				pkgRepository, repoErr := queryDebianRepository(ctx, pkg.Name, isPerconaPackage(pkgNamePattern))
				if repoErr != nil {
					zap.L().Sugar().Warnw("failed to get package repository info", zap.Error(repoErr), zap.String("package", pkg.Name))
					// go to next package silently
					continue
				}
				pkg.Repository = *pkgRepository
			}
		}
		toReturn = append(toReturn, pkgL...)
	}
	return toReturn
}

func isPerconaPackage(packageNamePattern string) bool {
	if len(packageNamePattern) == 0 {
		return false
	}

	perconaPkgList := append(getCommonPerconaPackages(), getDebianPerconaPackages()...)
	for _, pkgPattern := range perconaPkgList {
		if packageNamePattern == pkgPattern {
			return true
		}
	}
	return false
}

// getCommonPerconaPackages returns list of Percona package patterns that have the same names both on Debian and RHEL systems.
func getCommonPerconaPackages() []string {
	return []string{
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
