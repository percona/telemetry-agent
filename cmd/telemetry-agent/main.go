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
	"flag"
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

	platformClient "github.com/percona-lab/telemetry-agent/platform"

	"github.com/percona-lab/telemetry-agent/config"
	"github.com/percona-lab/telemetry-agent/logger"
	"github.com/percona-lab/telemetry-agent/metrics"
	"github.com/percona-lab/telemetry-agent/utils"
)

// Creates the minimum required directory structure for Telemetry Agent functionality.
func createTelemetryDirs(c config.Config) error {
	d := c.TelemetryHistoryPath
	zap.L().Sugar().Debugw("checking/creating telemetry directory", zap.String("directory", d))

	err := os.MkdirAll(d, os.ModePerm)
	if err != nil {
		zap.L().Sugar().Errorw("can't create directory",
			zap.String("directory", d),
			zap.Error(err))
		return err
	}
	return nil
}

// Create Percona Platform HTTP client for sending telemetry reports.
func createPerconaPlatformClient(c config.Config) (*platformClient.Client, error) {
	u, err := url.ParseRequestURI(c.PerconaTelemetryURL)
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
		platformClient.WithResendTimeout(time.Second*time.Duration(c.TelemetryResendTimeout)),
		platformClient.WithRetryCount(5)), nil
}

func processPillarsMetrics(c config.Config) []*metrics.File {
	l := zap.L().Sugar()

	pillarMetrics := make([]*metrics.File, 0, 1)

	l.Infow("processing PS metrics", zap.String("directory", c.PSMetricsPath))
	if pMetrics, err := metrics.ProcessPSMetrics(c.PSMetricsPath); err != nil {
		l.Errorw("failed to process PS metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}

	l.Infow("processing PXC metrics", zap.String("directory", c.PXCMetricsPath))
	if pMetrics, err := metrics.ProcessPXCMetrics(c.PXCMetricsPath); err != nil {
		l.Errorw("failed to process PXC metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}

	l.Infow("processing PSMDB metrics", zap.String("directory", c.PSMDBMetricsPath))
	if pMetrics, err := metrics.ProcessPSMDBMetrics(c.PSMDBMetricsPath); err != nil {
		l.Errorw("failed to process PSMDB metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}

	l.Infow("processing PG metrics", zap.String("directory", c.PGMetricsPath))
	if pMetrics, err := metrics.ProcessPGMetrics(c.PGMetricsPath); err != nil {
		l.Errorw("failed to process PG metrics", zap.Error(err))
	} else {
		pillarMetrics = append(pillarMetrics, pMetrics...)
	}
	return pillarMetrics
}

// The main function for processing Percona Pillar's telemetry and sending it to Percona Platform.
func processMetrics(ctx context.Context, c config.Config, platformClient *platformClient.Client) { //nolint:cyclop
	l := zap.L().Sugar()

	l.Info("scraping host metrics")
	hostMetrics, err := metrics.ScrapeHostMetrics()
	if err != nil {
		l.Warnw("failed to scrape host metrics", zap.Error(err))
	}
	hostInstanceID := hostMetrics.Metrics[metrics.InstanceIDKey]
	// instanceId is not needed in main metrics set
	delete(hostMetrics.Metrics, metrics.InstanceIDKey)

	l.Info("scraping installed Percona packages metrics")
	packages, err := metrics.ScrapeInstalledPackages(ctx)
	if err != nil {
		l.Warnw("failed to list installed packages", zap.Error(err))
	}

	if len(packages) != 0 {
		// add info about installed packages to host metrics.
		jsonData, err := json.Marshal(packages)
		if err != nil {
			l.Warnw("failed to marshal installed packages into JSON", zap.Error(err))
		}
		hostMetrics.Metrics["installed_packages"] = string(jsonData)
	}

	pillarMetrics := processPillarsMetrics(c)
	for _, pillarM := range pillarMetrics {
		// prepare request to Percona Platform
		reportMetrics := make([]*platformReporter.GenericReport_Metric, 0, 1)

		// copy pillar metrics to Platform request
		for k, v := range pillarM.Metrics {
			reportMetrics = append(reportMetrics, &platformReporter.GenericReport_Metric{
				Key:   k,
				Value: v,
			})
		}

		// enrich Platform request with host metrics
		for k, v := range hostMetrics.Metrics {
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
			if !errors.Is(err, context.Canceled) {
				metricsLogger.Errorw("error during sending telemetry", zap.Error(err))
			}
			continue
		}

		// write sent data to history file
		historyFile := filepath.Join(c.TelemetryHistoryPath, filepath.Base(pillarM.Filename))
		l.Infow("writing metrics to history file",
			zap.String("pillar file", pillarM.Filename),
			zap.String("history file", historyFile))
		if err := metrics.WriteMetricsToHistory(historyFile, report); err != nil {
			l.Errorw("failed to write metrics into history file",
				zap.String("pillar file", pillarM.Filename),
				zap.String("history file", historyFile),
				zap.Error(err))
			continue
		}

		// remove original Pillar's metrics file
		l.Infow("removing metrics file", zap.String("file", pillarM.Filename))
		if err := os.Remove(pillarM.Filename); err != nil {
			l.Errorw("failed to remove metrics file",
				zap.String("file", pillarM.Filename),
				zap.Error(err))
			continue
		}
	}
}

func main() {
	verboseF := flag.Bool("verbose", false, "enable verbose logging")
	devModeF := flag.Bool("dev-mode", false, "enable development mode in logging")
	flag.Usage = func() {
		flagSet := flag.CommandLine
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		order := []string{"verbose"}
		for _, name := range order {
			f := flagSet.Lookup(name)
			fmt.Fprintf(os.Stderr, "-%s\n\t%s\n", f.Name, f.Usage)
		}
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, `Environment variables:

PERCONA_TELEMETRY_ROOT_PATH - define Percona telemetry root path on local filesystem. Default: /usr/local/percona/telemetry
PERCONA_TELEMETRY_CHECK_INTERVAL - define time interval in seconds for checking Percona Pillars telemetry. Default: 86400 sec
PERCONA_TELEMETRY_URL - define Percona Platform URL for sending Pillars telemetry to. Default: https://check.percona.com/v1/telemetry/GenericReport
PERCONA_TELEMETRY_RESEND_INTERVAL - define wait time in seconds to sleep before retrying request to Percona Platform in case of request failure. Default: 60 sec
PERCONA_TELEMETRY_HISTORY_KEEP_INTERVAL - define time interval in seconds for keeping old history telemetry files on filesystem. Default: 604800 sec (7d)

`)
	}
	flag.Parse()

	logger.SetupGlobal(&logger.GlobalOpts{LogName: "telemetry-agent", LogDevMode: *devModeF, LogDebug: *verboseF})
	l := zap.L().Sugar()
	defer func(l *zap.SugaredLogger) {
		_ = l.Sync()
	}(l)

	l.Info("parsing env params")
	conf := config.InitConfig()
	l.Infow("values from config:", zap.Any("config", conf))

	// check that <telemetry root>/history dir exists on filesystem
	if err := createTelemetryDirs(conf); err != nil {
		l.DPanic(err)
	}

	pltClient, err := createPerconaPlatformClient(conf)
	if err != nil {
		l.DPanic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l.Info("Percona Telemetry Agent started")
	var wg sync.WaitGroup
	wg.Add(1)
	utils.SignalRunner(
		func() {
			checkIntv := time.Duration(conf.TelemetryCheckInterval) * time.Second
			l.Infof("sleeping for %d seconds before first iteration", conf.TelemetryCheckInterval)

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

					l.Infow("cleaning up history metric files", zap.String("directory", conf.TelemetryHistoryPath))
					if err := metrics.CleanupMetricsHistory(conf.TelemetryHistoryPath, conf.TelemetryHistoryKeepInterval); err != nil {
						l.Errorw("error during history metric directory cleanup", zap.Error(err))
						// not critical error, keep processing
					}

					l.Info("processing Pillars metric files")
					processMetrics(ctx, conf, pltClient)
					l.Info(fmt.Sprintf("sleep for %d seconds", conf.TelemetryCheckInterval))
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
