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

// Package main is the entry point of the service
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	platformReporter "github.com/percona-platform/saas/gen/telemetry/generic"
	platformLogger "github.com/percona-platform/saas/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	platformClient "github.com/percona/telemetry-agent/platform"

	"github.com/percona/telemetry-agent/config"
	"github.com/percona/telemetry-agent/logger"
	"github.com/percona/telemetry-agent/metrics"
	"github.com/percona/telemetry-agent/utils"
)

// Creates the minimum required directory structure for Telemetry Agent functionality.
func createTelemetryDirs(dirs ...string) error {
	const historyDirPermissions = 0o775

	for _, d := range dirs {
		zap.L().Sugar().Debugw("checking/creating telemetry directory", zap.String("directory", d))

		cleanPath := filepath.Clean(d)
		if _, err := os.Stat(cleanPath); err != nil {
			if !os.IsNotExist(err) {
				return err
			}

			if err = os.MkdirAll(d, os.ModeDir|historyDirPermissions); err != nil {
				zap.L().Sugar().Errorw("can't create directory",
					zap.String("directory", d),
					zap.Error(err))
				return err
			}
		}
	}
	return nil
}

// Create Percona Platform HTTP client for sending telemetry reports.
func createPerconaPlatformClient(c config.Config) (*platformClient.Client, error) {
	u, err := url.ParseRequestURI(c.Platform.URL)
	if err != nil {
		return nil, fmt.Errorf("can't create Percona Platform client: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, errors.New("invalid Percona Platform Telemetry URL: scheme or host is missed")
	}

	return platformClient.New(
		platformClient.WithLogger(zap.L().Named("perconaPlatformClient").Sugar()),
		platformClient.WithBaseURL(u.Scheme+"://"+u.Host),
		platformClient.WithLogFullRequest(),
		platformClient.WithResendTimeout(time.Second*time.Duration(c.Platform.ResendTimeout)),
		platformClient.WithRetryCount(5),
		platformClient.WithClientTimeout(60*time.Second)), nil
}

func processPillarsMetrics(c config.Config) []*metrics.File {
	l := zap.L().Sugar()

	pillarMetrics := make([]*metrics.File, 0, 1)

	l.Infow("processing PS metrics", zap.String("directory", c.Telemetry.PSMetricsPath))
	if pMetrics, err := metrics.ProcessPSMetrics(c.Telemetry.PSMetricsPath); err != nil {
		l.Warnw("failed to process PS metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}

	l.Infow("processing PXC metrics", zap.String("directory", c.Telemetry.PXCMetricsPath))
	if pMetrics, err := metrics.ProcessPXCMetrics(c.Telemetry.PXCMetricsPath); err != nil {
		l.Warnw("failed to process PXC metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}

	l.Infow("processing PSMDB metrics", zap.String("directory", c.Telemetry.PSMDBMetricsPath))
	if pMetrics, err := metrics.ProcessPSMDBMetrics(c.Telemetry.PSMDBMetricsPath); err != nil {
		l.Warnw("failed to process PSMDB metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}

	l.Infow("processing PG metrics", zap.String("directory", c.Telemetry.PGMetricsPath))
	if pMetrics, err := metrics.ProcessPGMetrics(c.Telemetry.PGMetricsPath); err != nil {
		l.Warnw("failed to process PG metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}
	return pillarMetrics
}

// The main function for processing Percona Pillar's telemetry and sending it to Percona Platform.
func processMetrics(ctx context.Context, c config.Config, platformClient *platformClient.Client) { //nolint:cyclop
	l := zap.L().Sugar()

	pillarMetrics := processPillarsMetrics(c)
	if len(pillarMetrics) == 0 {
		l.Info("no Pillar metrics files found, skip scraping host metrics and sending telemetry")
		return
	}

	l.Info("scraping host metrics")
	hostMetrics := metrics.ScrapeHostMetrics(ctx)
	hostInstanceID := hostMetrics.Metrics[metrics.InstanceIDKey]
	// instanceId is not needed in main metrics set
	delete(hostMetrics.Metrics, metrics.InstanceIDKey)

	l.Info("scraping installed Percona packages")
	if installedPackages := metrics.ScrapeInstalledPackages(ctx); len(installedPackages) != 0 {
		// add info about installed packages to host metrics.
		if jsonData, err := json.Marshal(installedPackages); err != nil {
			l.Warnw("failed to marshal installed Percona packages into JSON, skip it", zap.Error(err))
		} else {
			hostMetrics.Metrics["installed_packages"] = string(jsonData)
		}
	}

	for _, pillarM := range pillarMetrics {
		// prepare request to Percona Platform
		reportMetrics := make([]*platformReporter.GenericReport_Metric, 0, 1)

		// copy host metrics to Platform request
		for k, v := range hostMetrics.Metrics {
			reportMetrics = append(reportMetrics, &platformReporter.GenericReport_Metric{
				Key:   k,
				Value: v,
			})
		}

		// copy pillar metrics to Platform request
		for k, v := range pillarM.Metrics {
			reportMetrics = append(reportMetrics, &platformReporter.GenericReport_Metric{
				Key:   k,
				Value: v,
			})
		}

		report := &platformReporter.ReportRequest{
			Reports: []*platformReporter.GenericReport{
				{
					Id:            uuid.New().String(), // each request shall have unique ID
					CreateTime:    timestamppb.New(pillarM.Timestamp),
					InstanceId:    hostInstanceID,
					ProductFamily: pillarM.ProductFamily,
					Metrics:       reportMetrics,
				},
			},
		}

		metricsLogger := l.With(zap.String("file", pillarM.Filename))
		platformCtx := platformLogger.GetContextWithLogger(ctx, metricsLogger.Desugar())
		// send request to Percona Platform
		if err := platformClient.SendTelemetry(platformCtx, "", report); err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				// main process loop is terminated, no need to continue.
				// we can't continue this particular metrics file processing because we don't know what was sent and what was not.
				// try to send this metrics file again on next iteration.
				return
			default:
				// any other errors during sending data (including request timeout).
				// we can't continue this particular metrics file processing because we don't know what was sent and what was not.
				// try to send this metrics file again on next iteration.
				// pass over to next metrics file.
				metricsLogger.Warnw("error during sending telemetry, will try on next iteration", zap.Error(err))
				continue
			}
		}

		// write sent data to history file
		historyFile := filepath.Join(c.Telemetry.HistoryPath, filepath.Base(pillarM.Filename))
		l.Infow("writing metrics to history file",
			zap.String("pillar file", pillarM.Filename),
			zap.String("history file", historyFile))
		if err := metrics.WriteMetricsToHistory(historyFile, report); err != nil {
			l.Errorw("failed to write metrics into history file, will try on next iteration",
				zap.String("pillar file", pillarM.Filename),
				zap.String("history file", historyFile),
				zap.Error(err))
			continue
		}

		// remove original Pillar's metrics file
		l.Infow("removing metrics file", zap.String("file", pillarM.Filename))
		if err := os.Remove(pillarM.Filename); err != nil {
			l.Errorw("failed to remove metrics file, will try on next iteration",
				zap.String("file", pillarM.Filename),
				zap.Error(err))
			continue
		}
	}
}

func main() {
	conf := config.InitConfig()
	if conf.Version {
		fmt.Fprintf(os.Stdout, "Version: %s\n", config.Version)
		fmt.Fprintf(os.Stdout, "Commit: %s\n", config.Commit)
		fmt.Fprintf(os.Stdout, "Build date: %s\n", config.BuildDate)
		os.Exit(0)
	}

	logger.SetupGlobal(&logger.GlobalOpts{LogName: "telemetry-agent", LogDevMode: conf.Log.DevMode, LogDebug: conf.Log.Verbose})
	l := zap.L().Sugar()
	defer func(l *zap.SugaredLogger) {
		_ = l.Sync()
	}(l)

	l.Infow("values from config:", zap.Any("config", conf))

	// check that <telemetry root>/history dir exists on filesystem
	if err := createTelemetryDirs(conf.Telemetry.HistoryPath); err != nil {
		l.Panic(err)
	}

	pltClient, err := createPerconaPlatformClient(conf)
	if err != nil {
		l.Panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l.Info("Percona Telemetry Agent started")
	var wg sync.WaitGroup
	wg.Add(1)
	utils.SignalRunner(
		func() {
			checkIntv := time.Duration(conf.Telemetry.CheckInterval) * time.Second
			l.Infof("sleeping for %d seconds before first iteration", conf.Telemetry.CheckInterval)

			ticker := time.NewTicker(checkIntv)
			for {
				select {
				case <-ctx.Done():
					// terminate program
					l.Infow("terminating main loop")
					ticker.Stop()
					wg.Done()
					return
				case <-ticker.C:
					// start new metrics processing iteration
					l.Info("start metrics processing iteration")

					l.Infow("cleaning up history metric files", zap.String("directory", conf.Telemetry.HistoryPath))
					if err := metrics.CleanupMetricsHistory(conf.Telemetry.HistoryPath, conf.Telemetry.HistoryKeepInterval); err != nil {
						l.Errorw("error during history metrics directory cleanup", zap.Error(err))
						// not critical error, keep processing
					}

					l.Info("processing Pillars metrics files")
					processMetrics(ctx, conf, pltClient)
					l.Info(fmt.Sprintf("sleep for %d seconds", conf.Telemetry.CheckInterval))
				}
			}
		},
		func() {
			cancel()
			wg.Wait()
		},
	)
	l.Info("finished")
}
