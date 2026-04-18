package config

import (
	"flag"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

// Config holds all service configuration.
type Config struct {
	Port              int
	Host              string
	APIKey            string // comma-separated keys, JSON file path, or inline JSON
	Timeout           int    // seconds
	LogLevel          string
	ShowThinking      bool   // default: show thinking process for thinking models
	CORSOrigins       string
	RateLimit         int    // global rate limit: max requests per minute per key (0 = disabled)
	IPWhitelist       string // comma-separated IPs/CIDRs; default "127.0.0.1,::1" (loopback only); set "" or "0.0.0.0/0,::/0" to allow all
	TrustedProxyCount int    // number of trusted reverse proxies (0 = trust none, use RemoteAddr)
	ModelRefresh      int    // model list refresh interval in seconds (0 = disable auto-refresh)
}

const (
	defaultPort         = 39527
	defaultTimeout      = 120
	defaultRateLimit    = 0
	defaultModelRefresh = 300
	minPort             = 1
	maxPort             = 65535
)

// FallbackModels is the hardcoded model list used when upstream fetch fails.
var FallbackModels = []string{
	"qwen3-vl-30b-a3b-thinking",
	"qwen3-vl-30b-a3b-instruct",
	"qwen3-8b",
	"minimax-m2p5",
	"llama-v3p3-70b-instruct",
	"kimi-k2p5",
	"gpt-oss-20b",
	"gpt-oss-120b",
	"glm-5",
	"glm-4p7",
	"deepseek-v3p2",
	"deepseek-v3p1",
}

// Load reads configuration from environment variables only.
// Command-line flags should be parsed separately and passed via ApplyFlags.
func Load() *Config {
	cfg := &Config{}

	// Defaults
	cfg.Port = defaultPort
	cfg.Host = ""
	cfg.APIKey = "sk-admin"
	cfg.Timeout = defaultTimeout
	cfg.LogLevel = "info"
	cfg.ShowThinking = false
	cfg.CORSOrigins = "*"
	cfg.RateLimit = defaultRateLimit  // disabled by default; set >0 to enable
	cfg.IPWhitelist = "127.0.0.1,::1" // default: loopback only
	cfg.TrustedProxyCount = 0         // default: trust no proxy, use RemoteAddr
	cfg.ModelRefresh = defaultModelRefresh

	// Environment variables
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n >= minPort && n <= maxPort {
				cfg.Port = n
			} else {
				slog.Warn("PORT out of range, using default", "value", v, "min", minPort, "max", maxPort)
			}
		} else {
			slog.Warn("invalid PORT value, using default", "value", v)
		}
	}
	if v := os.Getenv("HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Timeout = n
		} else if n <= 0 {
			slog.Warn("TIMEOUT must be positive, using default", "value", v)
		} else {
			slog.Warn("invalid TIMEOUT value, using default", "value", v)
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("SHOW_THINKING"); v != "" {
		cfg.ShowThinking = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		cfg.CORSOrigins = v
	}
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n >= 0 {
				cfg.RateLimit = n
			} else {
				slog.Warn("RATE_LIMIT must be non-negative, using default", "value", v)
			}
		} else {
			slog.Warn("invalid RATE_LIMIT value, using default", "value", v)
		}
	}
	if v, ok := os.LookupEnv("IP_WHITELIST"); ok {
		cfg.IPWhitelist = v
	}
	if v := os.Getenv("TRUSTED_PROXY_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.TrustedProxyCount = n
		} else {
			slog.Warn("invalid TRUSTED_PROXY_COUNT value, using default", "value", v)
		}
	}
	if v := os.Getenv("MODEL_REFRESH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.ModelRefresh = n
		} else {
			slog.Warn("invalid MODEL_REFRESH value, using default", "value", v)
		}
	}

	return cfg
}

// ApplyFlags parses command-line flags and overrides config values.
// This is called from main() to avoid flag pollution in tests.
func (c *Config) ApplyFlags(args []string) {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.IntVar(&c.Port, "port", c.Port, "listen port")
	fs.StringVar(&c.Host, "host", c.Host, "listen host (default: all interfaces)")
	fs.StringVar(&c.APIKey, "api-key", c.APIKey, "API key for authentication")
	fs.IntVar(&c.Timeout, "timeout", c.Timeout, "upstream request timeout in seconds")
	fs.StringVar(&c.LogLevel, "log-level", c.LogLevel, "log level: debug, info, warn, error")
	fs.BoolVar(&c.ShowThinking, "show-thinking", c.ShowThinking, "show thinking process for thinking models")
	fs.StringVar(&c.CORSOrigins, "cors-origins", c.CORSOrigins, "allowed CORS origins (comma-separated, * for all)")
	fs.IntVar(&c.RateLimit, "rate-limit", c.RateLimit, "max requests per minute per key (0 to disable)")
	fs.StringVar(&c.IPWhitelist, "ip-whitelist", c.IPWhitelist, "allowed IPs/CIDRs (comma-separated, empty to allow all)")
	fs.IntVar(&c.TrustedProxyCount, "trusted-proxy-count", c.TrustedProxyCount, "number of trusted reverse proxies for X-Forwarded-For (0 = trust none)")
	fs.IntVar(&c.ModelRefresh, "model-refresh", c.ModelRefresh, "model list refresh interval in seconds (0 to disable)")
	_ = fs.Parse(args[1:])
	if c.Port < minPort || c.Port > maxPort {
		slog.Warn("port out of range, using default", "value", c.Port, "min", minPort, "max", maxPort)
		c.Port = defaultPort
	}
	if c.Timeout <= 0 {
		slog.Warn("timeout must be positive, using default", "value", c.Timeout)
		c.Timeout = defaultTimeout
	}
	if c.RateLimit < 0 {
		slog.Warn("rate-limit must be non-negative, clamping to 0", "value", c.RateLimit)
		c.RateLimit = defaultRateLimit
	}
	if c.TrustedProxyCount < 0 {
		slog.Warn("trusted-proxy-count must be non-negative, clamping to 0", "value", c.TrustedProxyCount)
		c.TrustedProxyCount = 0
	}
	if c.ModelRefresh < 0 {
		slog.Warn("model-refresh must be non-negative, clamping to 0", "value", c.ModelRefresh)
		c.ModelRefresh = 0
	}
}

// Addr returns the listen address string.
func (c *Config) Addr() string {
	if c.Host != "" {
		return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	}
	return ":" + strconv.Itoa(c.Port)
}
