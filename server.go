package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

type Server struct {
	http.Server

	apiKey   string
	dataPath string
}

type UserModelMetrics struct {
	CreatedAt time.Time `json:"created_at"`
	Model     string    `json:"model"`
	UserId    string    `json:"user_id,omitempty"`
	UserName  string    `json:"user_name,omitempty"`
	api.Metrics
}

func NewServer(ctx context.Context, host string, port int, dataPath string, apiKey string) *Server {
	addr := fmt.Sprintf("%s:%d", host, port)
	mux := http.NewServeMux()
	server := &Server{
		http.Server{
			Addr:    addr,
			Handler: mux,
			BaseContext: func(l net.Listener) context.Context {
				// use a new context with additional variable available in the context
				// under a given key.
				//ctx = context.WithValue(ctx, "the_key", l.Addr().String())
				return ctx
			},
		},
		apiKey,
		dataPath,
	}
	mux.HandleFunc("/user-model-metrics", server.userModelMetricHandle)
	return server
}

func (s *Server) Run() {
	slog.Info(fmt.Sprintf("Server listening at %s", s.Addr))
	serverErr := s.ListenAndServe()
	if !errors.Is(serverErr, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", serverErr)
	} else {
		slog.Info("Server stopped")
	}
}

// authRequestHandler checks request for authorization details and
// returns true when request is authorized.
func (s *Server) authRequestHandle(w http.ResponseWriter, r *http.Request) bool {
	if s.requireApiKeyAuthorization() {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Unauthorized: Missing Authorization header")
			slog.Info("Unauthorized: Missing Authorization header")
			return false
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Unauthorized: Invalid Authorization header format")
			slog.Info("Unauthorized: Invalid Authorization header format")
			return false
		}

		apiKey := parts[1]

		if !s.isValidAPIKey(apiKey) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Unauthorized: Invalid API key")
			slog.Info("Unauthorized: Invalid API key")
			return false
		}
	}

	return true
}

// requireApiKeyAuthorization checks if authentication with API key is required.
func (s *Server) requireApiKeyAuthorization() bool {
	return len(s.apiKey) > 0
}

// isValidAPIKey checks if the provided API key is valid.
func (s *Server) isValidAPIKey(apiKey string) bool {
	return s.apiKey == strings.TrimSpace(apiKey)
}

func (s *Server) userModelMetricHandle(response http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	buffer := make([]byte, 2*1024)
	n, err := request.Body.Read(buffer)
	if (err != nil && err != io.EOF) || n == 0 {
		slog.Error("Failed to read payload", "error", err)
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(response, "Failed to read payload")
		return
	}

	userModelMetrics, err := parseUserModelMetrics(buffer[:n])
	if userModelMetrics == nil {
		slog.Error("Failed to read payload", "error", err)
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(response, fmt.Sprintf("Failed to parse payload: %s", err))
		return
	}

	s.saveUserModelMetrics(userModelMetrics)
}

func (s *Server) saveUserModelMetrics(userModelMetrics *UserModelMetrics) error {
	buffer, err := json.Marshal(userModelMetrics)
	if err != nil {
		slog.Error("Failed to marschal user model metrics", "error", err)
		return err
	}

	now := time.Now()
	nowYear, nowWeek := now.ISOWeek()
	outPath := path.Join(s.dataPath, "UserModelMetrics", fmt.Sprintf("%d-%d", nowYear, nowWeek), fmt.Sprintf("%s.json", now.Format(time.RFC3339Nano)))
	outDir := path.Dir(outPath)

	if _, err := os.Stat(outDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
				slog.Error("Failed to create directory for user model metrics", "error", err)
				return err
			}
		}
	}

	err = os.WriteFile(outPath, buffer, 0644)
	if err != nil {
		slog.Error("Failed to write user model metrics to file", "error", err)
		return err
	}

	slog.Info("Wrote user model metrics to file", "path", outPath)
	return nil
}

func parseUserModelMetrics(data []byte) (*UserModelMetrics, error) {
	userModelMetrics := UserModelMetrics{}

	slog.Debug("Parsing user model metrics payload", "data", string(data))
	if err := json.Unmarshal(data, &userModelMetrics); err != nil {
		slog.Error("Failed to extract user model metrics payload", "error", err)
		return nil, err
	} else {
		return &userModelMetrics, nil
	}
}
