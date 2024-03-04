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

// Package utils provides various helper functions.
package utils

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// SignalRunner runs a runner function until an interrupt signal is received, at which point it
// will call stopper.
func SignalRunner(runner, stopper func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		runner()
	}()

	s := <-signals
	zap.L().Sugar().Infof("Received signal: %s, shutdown", s)
	signal.Stop(signals)
	stopper()
}
