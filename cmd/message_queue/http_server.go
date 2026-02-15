package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func startHTTPServer(port string, logger *slog.Logger) *http.Server {
	// Start HTTP server for health checks// Start HTTP health server for k8s probes
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	srvHTTP := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		logger.Info("starting health HTTP server", "addr", ":"+port)
		if err := srvHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
		}
	}()
	return srvHTTP
}
func shutdownHTTPServer(srv *http.Server, logger *slog.Logger) {
	// shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	} else {
		logger.Info("http server shutdown gracefully")
	}
}
