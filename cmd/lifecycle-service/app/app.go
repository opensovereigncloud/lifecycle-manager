// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/ironcore-dev/lifecycle-manager/internal/service"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type LogFormat string

const (
	JSON LogFormat = "json"
	Text LogFormat = "text"
)

var logLevelMapping = map[string]slog.Leveler{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

type Options struct {
	kubeconfig string
	logLevel   string
	logFormat  string
	host       string
	port       int
	namespace  string
	// jobsConfig string
	horizon time.Duration
	workers uint64
	queue   uint64
	dev     bool
}

func (o *Options) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	fs.StringVar(&o.logLevel, "log-level", "info", "logging level")
	fs.StringVar(&o.logFormat, "log-format", "json", "logging format")
	fs.StringVar(&o.host, "host", "", "bind host")
	fs.IntVar(&o.port, "port", 8080, "bind port")
	fs.StringVar(&o.namespace, "namespace", "default", "default namespace name")
	// fs.StringVar(&o.jobsConfig, "jobs-configmap", "", "name of the config map containing jobs parameters")
	fs.DurationVar(&o.horizon, "horizon", time.Minute*30, "allowed lag for scan period check")
	fs.Uint64Var(&o.workers, "workers", 5, "number of workers to process tasks")
	fs.Uint64Var(&o.queue, "queue-capacity", 1024, "size of the scheduler's queue")
	fs.BoolVar(&o.dev, "dev", false, "development mode flag")
}

func Command() *cobra.Command {
	var opts Options

	cmd := &cobra.Command{
		Use: "lifecycle-service",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return Run(ctx, opts)
		},
	}

	fs := pflag.NewFlagSet("", 0)
	cmd.PersistentFlags().AddFlagSet(fs)
	opts.addFlags(cmd.Flags())

	return cmd
}

func Run(ctx context.Context, opts Options) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	srvOpts := service.Options{
		Cfg:           cfg,
		Log:           setupLogger(LogFormat(opts.logFormat), logLevelMapping[opts.logLevel], opts.dev),
		Host:          opts.host,
		Port:          opts.port,
		Namespace:     opts.namespace,
		Horizon:       opts.horizon,
		Workers:       opts.workers,
		QueueCapacity: opts.queue,
	}
	srv := service.NewGrpcServer(srvOpts)
	return srv.Start(ctx)
}

func setupLogger(format LogFormat, level slog.Leveler, dev bool) *slog.Logger {
	switch format {
	case JSON:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: dev,
			Level:     level,
		}))
	case Text:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: dev,
			Level:     level,
		}))
	}
	return nil
}
