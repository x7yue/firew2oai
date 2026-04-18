package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultModelsURL = "https://chat.fireworks.ai/models?function_calling=true"
	fetchTimeout     = 15 * time.Second
	maxResponseBytes = 2 << 20
)

var modelsURL = defaultModelsURL

func setModelsURL(u string) { modelsURL = u }

// ModelInfo holds metadata about a single upstream model.
type ModelInfo struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ContextLength int    `json:"contextLength"`
	SupportsTools bool   `json:"supportsTools"`
	SupportsVision bool  `json:"supportsVision"`
}

// upstreamModel is the JSON shape returned by chat.fireworks.ai/models.
type upstreamModel struct {
	Title         string `json:"title"`
	SupportsTools bool   `json:"supportsTools"`
	SupportsVision bool  `json:"supportsVision"`
	ContextLength int    `json:"contextLength"`
}

// upstreamResponse is the top-level JSON envelope.
type upstreamResponse struct {
	Models map[string]upstreamModel `json:"models"`
}

// Registry is a thread-safe, dynamically-refreshable model registry.
type Registry struct {
	mu       sync.RWMutex
	models   []ModelInfo
	modelSet map[string]bool

	client   *http.Client
	fallback []string
	stopCh   chan struct{}
	stopped  chan struct{}
}

// NewRegistry creates a Registry with the given fallback model list.
// The fallback is used if the first upstream fetch fails.
func NewRegistry(fallback []string, client *http.Client) *Registry {
	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}
	r := &Registry{
		client:   client,
		fallback: fallback,
		stopCh:   make(chan struct{}),
		stopped:  make(chan struct{}),
	}
	// Initialize with fallback so Valid/List work before first Refresh.
	r.setFallback()
	return r
}

// setFallback populates the registry from the hardcoded fallback list.
func (r *Registry) setFallback() {
	models := make([]ModelInfo, len(r.fallback))
	set := make(map[string]bool, len(r.fallback))
	for i, id := range r.fallback {
		models[i] = ModelInfo{ID: id, Title: id}
		set[id] = true
	}
	r.mu.Lock()
	r.models = models
	r.modelSet = set
	r.mu.Unlock()
}

// Refresh fetches the model list from upstream and updates the registry.
// Returns an error if the fetch fails; the existing list is preserved on failure.
func (r *Registry) Refresh(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var upstream upstreamResponse
	if err := json.Unmarshal(body, &upstream); err != nil {
		return fmt.Errorf("decode models: %w", err)
	}

	if len(upstream.Models) == 0 {
		return fmt.Errorf("upstream returned empty model list")
	}

	models := make([]ModelInfo, 0, len(upstream.Models))
	set := make(map[string]bool, len(upstream.Models))
	for id, m := range upstream.Models {
		models = append(models, ModelInfo{
			ID:             id,
			Title:          m.Title,
			ContextLength:  m.ContextLength,
			SupportsTools:  m.SupportsTools,
			SupportsVision: m.SupportsVision,
		})
		set[id] = true
	}

	r.mu.Lock()
	r.models = models
	r.modelSet = set
	r.mu.Unlock()

	slog.Info("model registry refreshed", "count", len(models))
	return nil
}

// StartAutoRefresh starts a background goroutine that refreshes the model list
// at the given interval. Pass interval <= 0 to disable.
func (r *Registry) StartAutoRefresh(interval time.Duration) {
	if interval <= 0 {
		close(r.stopped)
		return
	}
	go func() {
		defer close(r.stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stopCh:
				return
			case <-ticker.C:
				if err := r.Refresh(context.Background()); err != nil {
					slog.Warn("model registry refresh failed, keeping previous list", "error", err)
				}
			}
		}
	}()
}

// Stop stops the background auto-refresh goroutine.
func (r *Registry) Stop() {
	select {
	case <-r.stopCh:
		// already closed
	default:
		close(r.stopCh)
	}
	<-r.stopped
}

// Valid returns true if the model ID is in the current registry.
func (r *Registry) Valid(model string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modelSet[model]
}

// List returns a copy of the current model list.
func (r *Registry) List() []ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ModelInfo, len(r.models))
	copy(out, r.models)
	return out
}

// IsThinkingModel returns true if the model name contains "thinking".
func IsThinkingModel(model string) bool {
	return strings.Contains(model, "thinking")
}
