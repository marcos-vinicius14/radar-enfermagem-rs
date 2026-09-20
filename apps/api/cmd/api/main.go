package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
	internalhttp "github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/http"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/scheduler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, cfg.AppEnv)
	log.Info("starting radar-enfermagem-rs API",
		"env", cfg.AppEnv,
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
	)

	// Inicializa conexão com PostgreSQL (com timeout)
	dbCtx, dbCancel := context.WithTimeout(context.Background(), cfg.DBConnTimeout)
	defer dbCancel()

	db, err := database.New(dbCtx, cfg)
	if err != nil {
		log.Warn("could not connect to database on startup (will retry on readiness probe)", "error", err)
	} else {
		log.Info("database connection pool initialized successfully")
		defer db.Close()
	}

	var sched *scheduler.Scheduler
	if cfg.EnableScheduler && db != nil {
		httpClient := collector.NewResilientHTTPClient(collector.ResilientClientConfig{
			Timeout:           20 * time.Second,
			MaxRetries:        3,
			InitialRetryDelay: 200 * time.Millisecond,
			DefaultRPS:        cfg.CollectorRateLimitRPS,
			DefaultBurst:      cfg.CollectorRateLimitBurst,
		}, log)

		reg := collector.NewDefaultRegistry(httpClient, 20*time.Second)
		repo := database.NewJobRepository(db.Pool, log)
		svc := collector.NewService(repo, nil, nil, log)

		unknownThreshold := time.Duration(cfg.CollectorStatusUnknownHours) * time.Hour
		expiredThreshold := time.Duration(cfg.CollectorStatusExpiredHours) * time.Hour

		schedInstance, err := scheduler.NewScheduler(scheduler.Config{
			CronSchedule:     cfg.CollectorCronSchedule,
			Concurrency:      cfg.CollectorConcurrency,
			UnknownThreshold: unknownThreshold,
			ExpiredThreshold: expiredThreshold,
		}, svc, reg, log)
		if err != nil {
			log.Error("failed to create collector scheduler", "error", err)
		} else {
			if err := schedInstance.Start(context.Background()); err != nil {
				log.Error("failed to start collector scheduler", "error", err)
			} else {
				sched = schedInstance
				log.Info("collector scheduler started in background",
					"cron_schedule", cfg.CollectorCronSchedule,
					"concurrency", cfg.CollectorConcurrency,
				)
			}
		}
	}

	router := internalhttp.NewRouter(log, db)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Canal para escuta de sinais de encerramento do sistema
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		log.Error("server encountered fatal error", "error", err)
		if sched != nil {
			sched.Stop()
		}
		os.Exit(1)

	case sig := <-shutdown:
		log.Info("shutdown signal received, initiating graceful shutdown", "signal", sig.String())

		if sched != nil {
			sched.Stop()
		}

		// Contexto para graceful shutdown com timeout de 10 segundos
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("failed to shutdown HTTP server gracefully, forcing close", "error", err)
			_ = server.Close()
		}

		log.Info("server shutdown completed cleanly")
	}

}
