package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
)

// getLogLevel returns a log level
func getLogLevel() slog.Level {
	if envLevel, found := os.LookupEnv("WEBHOOK_LOG_LEVEL"); found {
		if strings.ToLower(envLevel) == "error" {
			return slog.LevelError
		}
		if strings.ToLower(envLevel) == "warn" || strings.ToLower(envLevel) == "warning" {
			return slog.LevelWarn
		}
		if strings.ToLower(envLevel) == "info" {
			return slog.LevelInfo
		}
		if strings.ToLower(envLevel) == "debug" {
			return slog.LevelDebug
		}
	}
	return slog.LevelInfo
}

// getLogJson returns whether to log in JSON format
func getLogJson() bool {
	if envFormat, found := os.LookupEnv("WEBHOOK_LOG_FORMAT"); found {
		if strings.ToLower(envFormat) == "json" {
			return true
		}
	}
	return false
}

// getHost returns the hostname to bind to
func getHost() string {
	var host = "0.0.0.0"
	if envHost, found := os.LookupEnv("WEBHOOK_HOST"); found {
		host = strings.TrimSpace(envHost)
	}
	return host
}

// getPort returns the port to bind to
func getPort() int {
	var port = 80
	if envPort, found := os.LookupEnv("WEBHOOK_PORT"); found {
		if p, err := strconv.Atoi(envPort); err == nil && p != 0 {
			port = p
		}
	}
	return port
}

// getDataPath extract data path from environment variable
func getDataPath() string {
	if envApiKey, found := os.LookupEnv("WEBHOOK_DATA_PATH"); found {
		return strings.TrimSpace(envApiKey)
	}
	return "data"
}

// getApiKey extract API key from environment variable
func getApiKey() string {
	if envApiKey, found := os.LookupEnv("WEBHOOK_API_KEY"); found {
		return strings.TrimSpace(envApiKey)
	}
	return ""
}

func initLogging(level slog.Level, logJson bool) {
	var logger *slog.Logger
	logOptions := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Use filename and line of the source
			if a.Key == slog.SourceKey {
				s := a.Value.Any().(*slog.Source)
				s.File = path.Base(s.File)
			}
			return a
		},
	}
	if logJson {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, logOptions))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, logOptions))
	}
	slog.SetDefault(logger)
}

func main() {
	initLogging(getLogLevel(), getLogJson())

	var host = getHost()
	var port = getPort()
	var dataPath = getDataPath()
	var apiKey = getApiKey()

	ctx, cancelCtx := context.WithCancel(context.Background())

	server := NewServer(ctx, host, port, dataPath, apiKey)
	go server.Run()
	serverUrl := fmt.Sprintf("http://%s", server.Addr)
	slog.Info(fmt.Sprintf("Webhook listening at %s", serverUrl))

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		slog.Info("Received", "signal", sig)
		done <- true
	}()

	// block until we receive the "done" via channel
	<-done
	cancelCtx()

	slog.Info("Shutdown server")
	if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
		slog.Error("Failed to shutdown server", "error", shutdownErr)
		return
	}
	slog.Info("Done.")
}
