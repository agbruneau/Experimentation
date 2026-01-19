// Package proxy provides HTTP reverse proxy for EDA-Lab services.
package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/edalab/pkg/observability"
)

// ServiceProxy forwards requests to backend services.
type ServiceProxy struct {
	simulatorURL string
	bancaireURL  string
	client       *http.Client
	logger       *slog.Logger
	metrics      *observability.Metrics
	service      string
}

// Config holds proxy configuration.
type Config struct {
	SimulatorURL string
	BancaireURL  string
	Timeout      time.Duration
}

// NewServiceProxy creates a new service proxy.
func NewServiceProxy(cfg Config, logger *slog.Logger, metrics *observability.Metrics, service string) *ServiceProxy {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ServiceProxy{
		simulatorURL: cfg.SimulatorURL,
		bancaireURL:  cfg.BancaireURL,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger:  logger,
		metrics: metrics,
		service: service,
	}
}

// ForwardToSimulator forwards a request to the Simulator service.
func (p *ServiceProxy) ForwardToSimulator(w http.ResponseWriter, r *http.Request, path string) {
	p.forward(w, r, p.simulatorURL, path, "simulator")
}

// ForwardToBancaire forwards a request to the Bancaire service.
func (p *ServiceProxy) ForwardToBancaire(w http.ResponseWriter, r *http.Request, path string) {
	p.forward(w, r, p.bancaireURL, path, "bancaire")
}

// forward proxies the request to the target service.
func (p *ServiceProxy) forward(w http.ResponseWriter, r *http.Request, baseURL, path, targetService string) {
	start := time.Now()

	// Build target URL
	targetURL, err := url.Parse(baseURL + path)
	if err != nil {
		p.logger.Error("failed to parse target URL",
			slog.String("base_url", baseURL),
			slog.String("path", path),
			slog.Any("error", err),
		)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Add query string
	targetURL.RawQuery = r.URL.RawQuery

	// Create proxy request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		p.logger.Error("failed to create proxy request", slog.Any("error", err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Copy headers (except Host)
	for key, values := range r.Header {
		if key == "Host" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Add forwarding headers
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)
	proxyReq.Header.Set("X-Forwarded-Proto", "http")

	// Execute request
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		p.logger.Error("proxy request failed",
			slog.String("target", targetURL.String()),
			slog.Any("error", err),
		)
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

	// Write status and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	// Log and record metrics
	duration := time.Since(start)
	p.logger.Debug("proxied request",
		slog.String("method", r.Method),
		slog.String("path", path),
		slog.String("target", targetService),
		slog.Int("status", resp.StatusCode),
		slog.Duration("duration", duration),
	)

	if p.metrics != nil {
		p.metrics.RecordHTTPRequest(p.service, r.Method, "/proxy/"+targetService, http.StatusText(resp.StatusCode), duration)
	}
}

// HealthCheck checks the health of backend services.
func (p *ServiceProxy) HealthCheck() map[string]string {
	status := make(map[string]string)

	// Check Simulator
	status["simulator"] = p.checkService(p.simulatorURL + "/api/v1/health")

	// Check Bancaire
	status["bancaire"] = p.checkService(p.bancaireURL + "/api/v1/health")

	return status
}

// checkService performs a health check on a service.
func (p *ServiceProxy) checkService(url string) string {
	ctx, cancel := p.client.Timeout, func() {}
	_ = ctx
	defer cancel()

	resp, err := p.client.Get(url)
	if err != nil {
		return "unhealthy"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "healthy"
	}
	return "unhealthy"
}
