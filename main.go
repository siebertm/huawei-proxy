package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up structured logging to stdout
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("config loaded",
		"inverter", cfg.InverterAddr(),
		"server_listen", cfg.Server.Listen,
		"web_listen", cfg.Web.Listen,
		"register_groups", len(cfg.RegisterGroups),
		"read_pause_ms", cfg.Polling.ReadPauseMs,
		"slow_interval_s", cfg.Polling.SlowIntervalS,
		"forward_unknown_reads", cfg.ForwardUnknownReads,
		"cache_path", cfg.CachePath,
		"log_level", cfg.LogLevel,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create cache
	cache, err := NewRegisterCache(cfg.CachePath)
	if err != nil {
		slog.Error("failed to open cache database", "path", cfg.CachePath, "error", err)
		os.Exit(1)
	}
	defer cache.Close()

	// Start web UI early so it's available during startup
	var web *WebServer
	if cfg.Web.Listen != "" {
		web = NewWebServer(cfg, cache)
		go func() {
			if err := web.ListenAndServe(ctx); err != nil {
				slog.Error("web server error", "error", err)
			}
		}()
	}

	// Connect to inverter
	slog.Info("connecting to inverter", "address", cfg.InverterAddr(), "unit_ids", cfg.Inverter.UnitIDs)
	inverterClient, err := NewInverterClient(cfg)
	if err != nil {
		slog.Error("failed to connect to inverter", "error", err)
		os.Exit(1)
	}
	defer inverterClient.Close()
	slog.Info("connected to inverter")

	// Create reader and do initial scan
	reader := NewReader(cfg, inverterClient, cache)

	slog.Info("performing initial register scan...")
	if err := reader.InitialScan(ctx); err != nil {
		slog.Error("initial scan failed", "error", err)
		os.Exit(1)
	}

	// Start reader loop — also makes stats available to the web UI
	if web != nil {
		web.SetReader(reader)
	}
	go reader.Run(ctx)

	// Start Modbus TCP server
	server := NewServer(cfg, cache, inverterClient)
	go func() {
		if err := server.ListenAndServe(ctx); err != nil {
			slog.Error("server error", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	slog.Info("received signal, shutting down", "signal", sig)
	cancel()
}
