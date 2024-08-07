package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/log"
	"github.com/kashalls/kromgo/pkg/kromgo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HealthCheckHandler returns the status of the service
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ReadinessHandler returns whether the service is ready to accept requests
func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Init initializes the http server
func Init(config configuration.KromgoConfig, serverConfig configuration.ServerConfig) (*http.Server, *http.Server) {

	mainRouter := chi.NewRouter()
	if serverConfig.ServerLogging {
		mainRouter.Use(middleware.Logger)
	}
	if serverConfig.RatelimitEnable {
		if serverConfig.RatelimitAll {
			mainRouter.Use(httprate.LimitAll(serverConfig.RatelimitRequestLimit, serverConfig.RatelimitWindowLength))
		} else if serverConfig.RatelimitByRealIP {
			mainRouter.Use(httprate.LimitByRealIP(serverConfig.RatelimitRequestLimit, serverConfig.RatelimitWindowLength))
		} else {
			mainRouter.Use(httprate.LimitByIP(serverConfig.RatelimitRequestLimit, serverConfig.RatelimitWindowLength))
		}
	}

	mainRouter.Get("/{metric}", func(w http.ResponseWriter, r *http.Request) {
		kromgo.KromgoRequestHandler(w, r, config)
	})

	mainServer := createHTTPServer(fmt.Sprintf("%s:%d", serverConfig.ServerHost, serverConfig.ServerPort), mainRouter, serverConfig.ServerReadTimeout, serverConfig.ServerWriteTimeout)
	go func() {
		log.Info("starting webhook server", zap.String("address", mainServer.Addr))
		if err := mainServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("unable to start webhook server", zap.String("address", mainServer.Addr), zap.Error(err))
		}
	}()

	healthRouter := chi.NewRouter()
	healthRouter.Get("/metrics", promhttp.Handler().ServeHTTP)
	healthRouter.Get("/healthz", HealthCheckHandler)
	healthRouter.Get("/-/health", HealthCheckHandler)
	healthRouter.Get("/readyz", ReadinessHandler)
	healthRouter.Get("/-/ready", ReadinessHandler)

	healthServer := createHTTPServer(fmt.Sprintf("%s:%d", serverConfig.HealthHost, serverConfig.HealthPort), healthRouter, serverConfig.ServerReadTimeout, serverConfig.ServerWriteTimeout)
	go func() {
		log.Info("starting health server", zap.String("address", healthServer.Addr))
		if err := healthServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("unable to start health server", zap.String("address", healthServer.Addr), zap.Error(err))
		}
	}()

	return mainServer, healthServer
}

func createHTTPServer(addr string, hand http.Handler, readTimeout, writeTimeout time.Duration) *http.Server {
	return &http.Server{
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		Addr:         addr,
		Handler:      hand,
	}
}

// ShutdownGracefully gracefully shutdown the http server
func ShutdownGracefully(mainServer *http.Server, healthServer *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh

	log.Info("shutting down servers due to received signal", zap.Any("signal", sig))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mainServer.Shutdown(ctx); err != nil {
		log.Error("error shutting down main server", zap.Error(err))
	}

	if err := healthServer.Shutdown(ctx); err != nil {
		log.Error("error shutting down health server", zap.Error(err))
	}
}
