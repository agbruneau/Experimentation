package proxy

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ServiceProxy proxies requests to backend services
type ServiceProxy struct {
	simulatorURL string
	bancaireURL  string
	client       *http.Client
	logger       *slog.Logger
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(simulatorURL, bancaireURL string, logger *slog.Logger) *ServiceProxy {
	return &ServiceProxy{
		simulatorURL: simulatorURL,
		bancaireURL:  bancaireURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// ForwardToSimulator forwards requests to the simulator service
func (p *ServiceProxy) ForwardToSimulator(w http.ResponseWriter, r *http.Request) {
	p.forward(w, r, p.simulatorURL)
}

// ForwardToBancaire forwards requests to the bancaire service
func (p *ServiceProxy) ForwardToBancaire(w http.ResponseWriter, r *http.Request) {
	p.forward(w, r, p.bancaireURL)
}

// forward forwards an HTTP request to the target URL
func (p *ServiceProxy) forward(w http.ResponseWriter, r *http.Request, targetURL string) {
	// Build target URL
	path := r.URL.Path
	url := targetURL + path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	p.logger.Debug("Forwarding request",
		slog.String("method", r.Method),
		slog.String("path", path),
		slog.String("target", url),
	)

	// Create new request
	req, err := http.NewRequestWithContext(r.Context(), r.Method, url, r.Body)
	if err != nil {
		p.logger.Error("Failed to create request", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy headers (except Host)
	for key, values := range r.Header {
		if strings.ToLower(key) == "host" {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("Request failed", slog.String("error", err.Error()))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy body
	io.Copy(w, resp.Body)
}

// HealthCheck checks if backend services are healthy
func (p *ServiceProxy) HealthCheck() map[string]bool {
	results := make(map[string]bool)

	// Check simulator
	results["simulator"] = p.checkHealth(p.simulatorURL + "/health")

	// Check bancaire
	results["bancaire"] = p.checkHealth(p.bancaireURL + "/health")

	return results
}

// checkHealth checks if a service is healthy
func (p *ServiceProxy) checkHealth(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
