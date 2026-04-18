package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRefresh_Success(t *testing.T) {
	upstream := upstreamResponse{
		Models: map[string]upstreamModel{
			"model-a": {Title: "Model A", ContextLength: 4096, SupportsTools: true},
			"model-b": {Title: "Model B", ContextLength: 8192, SupportsVision: true},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(upstream)
	}))
	defer srv.Close()

	reg := NewRegistry([]string{"fallback-model"}, srv.Client())
	origURL := modelsURL
	defer func() { setModelsURL(origURL) }()
	setModelsURL(srv.URL)

	if err := reg.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}

	if !reg.Valid("model-a") {
		t.Error("model-a should be valid after refresh")
	}
	if !reg.Valid("model-b") {
		t.Error("model-b should be valid after refresh")
	}
	if reg.Valid("fallback-model") {
		t.Error("fallback-model should not be valid after successful refresh")
	}

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("List() len = %d, want 2", len(list))
	}
}

func TestRefresh_Failure_KeepsPrevious(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reg := NewRegistry([]string{"fallback-model"}, srv.Client())
	origURL := modelsURL
	defer func() { setModelsURL(origURL) }()
	setModelsURL(srv.URL)

	err := reg.Refresh(context.Background())
	if err == nil {
		t.Fatal("Refresh() should fail on 500")
	}

	if !reg.Valid("fallback-model") {
		t.Error("fallback-model should still be valid after failed refresh")
	}
}

func TestRefresh_EmptyResponse_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(upstreamResponse{Models: map[string]upstreamModel{}})
	}))
	defer srv.Close()

	reg := NewRegistry([]string{"fallback-model"}, srv.Client())
	origURL := modelsURL
	defer func() { setModelsURL(origURL) }()
	setModelsURL(srv.URL)

	err := reg.Refresh(context.Background())
	if err == nil {
		t.Fatal("Refresh() should fail on empty model list")
	}

	if !reg.Valid("fallback-model") {
		t.Error("fallback-model should still be valid after empty refresh")
	}
}

func TestValid_BeforeRefresh_UsesFallback(t *testing.T) {
	reg := NewRegistry([]string{"fb-1", "fb-2"}, nil)
	if !reg.Valid("fb-1") {
		t.Error("fb-1 should be valid from fallback")
	}
	if reg.Valid("nonexistent") {
		t.Error("nonexistent should not be valid")
	}
}

func TestList_ReturnsCopy(t *testing.T) {
	reg := NewRegistry([]string{"m1", "m2"}, nil)
	list := reg.List()
	list[0].ID = "mutated"
	if reg.List()[0].ID == "mutated" {
		t.Error("List() should return a copy, not a reference")
	}
}

func TestIsThinkingModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"qwen3-vl-30b-a3b-thinking", true},
		{"some-thinking-model", true},
		{"deepseek-v3p2", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsThinkingModel(tt.model); got != tt.want {
			t.Errorf("IsThinkingModel(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestStartAutoRefresh_AndStop(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(upstreamResponse{
			Models: map[string]upstreamModel{
				"dynamic": {Title: "Dynamic"},
			},
		})
	}))
	defer srv.Close()

	reg := NewRegistry([]string{"fallback"}, srv.Client())
	origURL := modelsURL
	defer func() { setModelsURL(origURL) }()
	setModelsURL(srv.URL)

	reg.StartAutoRefresh(50 * time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	reg.Stop()

	if callCount == 0 {
		t.Error("auto-refresh should have called upstream at least once")
	}
	if !reg.Valid("dynamic") {
		t.Error("dynamic model should be valid after auto-refresh")
	}
}

func TestStartAutoRefresh_DisabledWithZero(t *testing.T) {
	reg := NewRegistry([]string{"fb"}, nil)
	reg.StartAutoRefresh(0)
	reg.Stop() // should not hang
}
