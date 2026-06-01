package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/kromgo"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/home-operations/kromgo/internal/server"
)

var (
	Version = "local"
	Gitsha  = "?"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "Path to the YAML config file")
	flag.Parse()

	initLogger()
	slog.Info("starting kromgo", "version", Version, "gitsha", Gitsha)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	sc, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("loading server config: %w", err)
	}

	prom, err := prometheus.New(cmp.Or(os.Getenv("PROMETHEUS_URL"), cfg.Prometheus), sc.QueryTimeout)
	if err != nil {
		return err
	}

	handler, err := kromgo.New(cfg, prom)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	return server.Run(ctx, sc, handler.Mux())
}

// initLogger configures the default slog logger from LOG_LEVEL and LOG_FORMAT.
func initLogger() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
