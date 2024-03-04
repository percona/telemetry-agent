/*
 * // Copyright (C) 2024 Percona LLC
 * //
 * // This program is free software: you can redistribute it and/or modify
 * // it under the terms of the GNU Affero General Public License as published by
 * // the Free Software Foundation, either version 3 of the License, or
 * // (at your option) any later version.
 * //
 * // This program is distributed in the hope that it will be useful,
 * // but WITHOUT ANY WARRANTY; without even the implied warranty of
 * // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * // GNU Affero General Public License for more details.
 * //
 * // You should have received a copy of the GNU Affero General Public License
 * // along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

package metrics

import (
	platformReporter "github.com/percona-platform/platform/gen/telemetry/generic"
)

// ProcessPSMetrics processes PS metrics and returns slice of *File.
// Each File corresponds to a separate metrics file.
func ProcessPSMetrics(path string) ([]*File, error) {
	return processMetricsDirectory(path, platformReporter.ProductFamily_PRODUCT_FAMILY_PS)
}

// ProcessPXCMetrics processes PXC metrics and returns slice of *File.
// Each File corresponds to a separate metrics file.
func ProcessPXCMetrics(path string) ([]*File, error) {
	return processMetricsDirectory(path, platformReporter.ProductFamily_PRODUCT_FAMILY_PXC)
}

// ProcessPSMDBMetrics processes PSMDB metrics and returns slice of *File.
// Each File corresponds to a separate metrics file.
func ProcessPSMDBMetrics(path string) ([]*File, error) {
	return processMetricsDirectory(path, platformReporter.ProductFamily_PRODUCT_FAMILY_PSMDB)
}

// ProcessPGMetrics processes PG metrics and returns slice of *File.
// Each File corresponds to a separate metrics file.
func ProcessPGMetrics(path string) ([]*File, error) {
	return processMetricsDirectory(path, platformReporter.ProductFamily_PRODUCT_FAMILY_POSTGRESQL)
}
