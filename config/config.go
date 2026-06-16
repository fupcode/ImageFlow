package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config stores the application configuration
type Config struct {
	// Server settings
	ServerAddr      string `json:"server_addr"`     // Server listen address
	ImageBasePath   string `json:"image_base_path"` // Base path for image storage
	AvifSupport     bool   `json:"avif_support"`    // Whether AVIF format is supported
	APIKey          string // API key for authentication
	MaxUploadCount  int    `json:"max_upload_count"` // Maximum number of images allowed in single upload
	ImageQuality    int    `json:"image_quality"`    // Image conversion quality (1-100)
	WorkerThreads   int    `json:"worker_threads"`   // Number of parallel worker threads
	Speed           int    `json:"speed"`            // Encoding speed (0-8, 0=slowest/highest quality)
	WorkerPoolSize  int    `json:"worker_pool_size"` // Size of worker pool for concurrent image processing
	DebugMode       bool   `json:"debug_mode"`       // Whether debug mode is enabled
	CleanupInterval int    `json:"cleanup_interval"` // Interval in minutes for cleaning expired images

	// Storage settings
	CustomDomain string `json:"custom_domain"` // Custom domain for public image URLs

	// Metadata storage settings
	MetadataSQLitePath string `json:"metadata_sqlite_path"` // SQLite database path for metadata
}

// GetBaseURL returns the base URL for image access.
func (c *Config) GetBaseURL() string {
	if c.CustomDomain != "" {
		baseURL := normalizePublicBaseURL(c.CustomDomain)
		return baseURL + "/images"
	}

	return "/images"
}

func normalizePublicBaseURL(rawURL string) string {
	baseURL := strings.TrimSuffix(strings.TrimSpace(rawURL), "/")
	if baseURL == "" {
		return ""
	}
	if strings.HasPrefix(baseURL, "http://") ||
		strings.HasPrefix(baseURL, "https://") ||
		strings.HasPrefix(baseURL, "//") {
		return baseURL
	}
	if strings.HasPrefix(baseURL, "localhost:") ||
		strings.HasPrefix(baseURL, "127.0.0.1:") ||
		strings.HasPrefix(baseURL, "0.0.0.0:") ||
		strings.HasPrefix(baseURL, "[::1]:") {
		return "http://" + baseURL
	}
	return "https://" + baseURL
}

// ClientConfig represents the configuration exposed to clients
type ClientConfig struct {
	MaxUploadCount int  `json:"maxUploadCount"` // Maximum number of images allowed per upload
	ImageQuality   int  `json:"imageQuality"`   // Image conversion quality (1-100)
	Speed          int  `json:"speed"`          // Encoding speed (0-8, 0=slowest/highest quality)
	AvifSupport    bool `json:"avifSupport"`    // Whether AVIF format is supported
}

// GetClientConfig returns configuration that can be exposed to clients
func (c *Config) GetClientConfig() ClientConfig {
	return ClientConfig{
		MaxUploadCount: c.MaxUploadCount,
		ImageQuality:   c.ImageQuality,
		Speed:          c.Speed,
		AvifSupport:    c.AvifSupport,
	}
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	// Default configuration
	cfg := &Config{
		ServerAddr:      "0.0.0.0:8686",
		ImageBasePath:   os.Getenv("LOCAL_STORAGE_PATH"),
		AvifSupport:     false,
		MaxUploadCount:  20,    // Default max upload: 20 images
		ImageQuality:    75,    // Default quality: 75
		WorkerThreads:   4,     // Default workers: 4 threads
		Speed:           5,     // Default speed: 5 (medium)
		WorkerPoolSize:  10,    // Default worker pool size: 10 concurrent tasks
		DebugMode:       false, // Default debug mode off
		CleanupInterval: 1,     // Default cleanup interval: 1 minute

		// Metadata store defaults
		MetadataSQLitePath: "data/metadata.db",
	}

	// If LOCAL_STORAGE_PATH is not set, use default value
	if cfg.ImageBasePath == "" {
		cfg.ImageBasePath = "static/images"
	}

	// Ensure path is relative
	if !filepath.IsAbs(cfg.ImageBasePath) {
		cfg.ImageBasePath = filepath.Join(".", cfg.ImageBasePath)
	}

	// Try to load .env file, but don't require it
	_ = godotenv.Load()

	// Load environment variables
	cfg.loadEnvVars()

	// If config file exists, load additional configuration from file
	if _, err := os.Stat("config/config.json"); err == nil {
		file, err := os.Open("config/config.json")
		if err != nil {
			return nil, err
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// loadEnvVars loads configuration from environment variables
func (c *Config) loadEnvVars() {
	// Server settings
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		c.ServerAddr = addr
	}
	c.APIKey = os.Getenv("API_KEY")

	// Debug mode
	if debug := os.Getenv("DEBUG_MODE"); debug != "" {
		c.DebugMode = debug == "true"
	}
	if avifEnabled := os.Getenv("AVIF_ENABLED"); avifEnabled != "" {
		c.AvifSupport = avifEnabled == "true"
	}

	if customDomain := os.Getenv("CUSTOM_DOMAIN"); customDomain != "" {
		c.CustomDomain = customDomain
	}

	// Parse integer environment variables
	envVarInt := map[string]*int{
		"MAX_UPLOAD_COUNT": &c.MaxUploadCount,
		"IMAGE_QUALITY":    &c.ImageQuality,
		"WORKER_THREADS":   &c.WorkerThreads,
		"SPEED":            &c.Speed,
		"WORKER_POOL_SIZE": &c.WorkerPoolSize,
		"CLEANUP_INTERVAL": &c.CleanupInterval,
	}

	for envName, ptr := range envVarInt {
		if val := os.Getenv(envName); val != "" {
			if num, err := strconv.Atoi(val); err == nil {
				*ptr = num
			}
		}
	}

	// Ensure speed is within valid range (0-8)
	if c.Speed < 0 {
		c.Speed = 0
	} else if c.Speed > 8 {
		c.Speed = 8
	}

	// Metadata store settings
	if sqlitePath := os.Getenv("METADATA_SQLITE_PATH"); sqlitePath != "" {
		c.MetadataSQLitePath = sqlitePath
	}
}
