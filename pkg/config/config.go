package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Agents    AgentsConfig           `json:"agents"`
	Providers map[string]ModelConfig `json:"providers"`
	Channels  ChannelsConfig         `json:"channels"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

type AgentDefaults struct {
	Workspace           string  `json:"workspace"`
	Model               string  `json:"model"`
	MaxTokens           int     `json:"maxTokens"`
	Temperature         float64 `json:"temperature"`
	MaxToolIterations   int     `json:"maxToolIterations"`
	MemoryWindow        int     `json:"memoryWindow"`
	RestrictToWorkspace bool    `json:"restrictToWorkspace"`
}

type ModelConfig struct {
	APIKey  string `json:"apiKey"`
	APIBase string `json:"apiBase,omitempty"`

	// internal fields
	Vendor string `json:"-"`
	Model  string `json:"-"`
}

type ChannelsConfig struct {
	Telegram TelegramConfig `json:"telegram"`
}

type TelegramConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"botToken"`
	AllowFrom []string `json:"allow_from"`
}

// Load loads the configuration with a 3-tier priority: Default -> JSON -> Env
func Load(configPath string) (*Config, error) {
	cfg := &Config{}

	// 1. Apply defaults
	applyDefaults(cfg)

	// Expand ~ to user home dir for configPath
	expandedPath := expandHome(configPath)

	// 2. Load from JSON if exists
	if _, err := os.Stat(expandedPath); err == nil {
		checkFilePermissions(expandedPath)

		data, err := os.ReadFile(expandedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		warnIfSensitiveDataPresent(data)
	}

	// 3. Apply Environment Variable overrides
	applyEnvOverrides(cfg)

	// Expand workspace path
	cfg.Agents.Defaults.Workspace = expandHome(cfg.Agents.Defaults.Workspace)

	// Check workspace directory isolation
	_ = checkWorkspaceIsolation(cfg.Agents.Defaults.Workspace)

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	cfg.Agents.Defaults.Workspace = DefaultWorkspace
	cfg.Agents.Defaults.Model = DefaultModel
	cfg.Agents.Defaults.MaxTokens = DefaultMaxTokens
	cfg.Agents.Defaults.Temperature = DefaultTemperature
	cfg.Agents.Defaults.MaxToolIterations = DefaultMaxToolIterations
	cfg.Agents.Defaults.RestrictToWorkspace = DefaultRestrictToWS
}

func loadEnvFile() {
	envPath := filepath.Join(filepath.Dir(os.Args[0]), ".env")
	if _, err := os.Stat(envPath); err != nil {
		envPath = ".env"
		if _, err := os.Stat(envPath); err != nil {
			return
		}
	}

	data, _ := os.ReadFile(envPath)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, `"'`)
			os.Setenv(key, value)
		}
	}
}

func applyEnvOverrides(cfg *Config) {
	loadEnvFile()

	if v := os.Getenv("MINIBOT_AGENTS_DEFAULTS_WORKSPACE"); v != "" {
		cfg.Agents.Defaults.Workspace = v
	}
	if v := os.Getenv("MINIBOT_AGENTS_DEFAULTS_MODEL"); v != "" {
		cfg.Agents.Defaults.Model = v
	}
	if v := os.Getenv("MINIBOT_AGENTS_DEFAULTS_MAX_TOKENS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			cfg.Agents.Defaults.MaxTokens = val
		}
	}
	if v := os.Getenv("MINIBOT_AGENTS_DEFAULTS_TEMPERATURE"); v != "" {
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Agents.Defaults.Temperature = val
		}
	}
	if v := os.Getenv("MINIBOT_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			cfg.Agents.Defaults.MaxToolIterations = val
		}
	}
	if v := os.Getenv("MINIBOT_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			cfg.Agents.Defaults.RestrictToWorkspace = val
		}
	}

	if v := os.Getenv("MINIBOT_PROVIDERS_MINIMAX_API_KEY"); v != "" {
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]ModelConfig)
		}
		if p, ok := cfg.Providers["minimax"]; ok {
			p.APIKey = v
			cfg.Providers["minimax"] = p
		} else {
			cfg.Providers["minimax"] = ModelConfig{APIKey: v, APIBase: "https://api.minimax.io/v1"}
		}
	}

	if v := os.Getenv("MINIBOT_CHANNELS_TELEGRAM_BOT_TOKEN"); v != "" {
		cfg.Channels.Telegram.Enabled = true
		cfg.Channels.Telegram.Token = v
	}
	if v := os.Getenv("MINIBOT_CHANNELS_TELEGRAM_ALLOW_FROM"); v != "" {
		cfg.Channels.Telegram.AllowFrom = strings.Split(v, ",")
	}
}

// FindModel returns the ModelConfig for the given model string (e.g., "minimax/MiniMax-M2.5").
func (c *Config) FindModel(modelDef string) (*ModelConfig, error) {
	parts := strings.SplitN(modelDef, "/", 2)
	vendor := "openai" // Default fallback

	if len(parts) == 2 {
		vendor = strings.ToLower(parts[0])
	}

	if p, ok := c.Providers[vendor]; ok {
		p.Vendor = vendor
		p.Model = modelDef // store the full model string to pass down to factory
		return &p, nil
	}

	return nil, fmt.Errorf("model not found: %s", modelDef)
}

// expandHome resolves "~" to the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

func warnIfSensitiveDataPresent(data []byte) {
	content := string(data)
	sensitiveKeys := []string{"apiKey", "botToken", "APikey", "token"}

	for _, key := range sensitiveKeys {
		if idx := strings.Index(content, `"`+key+`"`); idx != -1 {
			start := idx + len(key) + 3
			if start < len(content) {
				start := start
				for ; start < len(content) && (content[start] == ' ' || content[start] == ':'); start++ {
				}
				if start < len(content) && content[start] == '"' {
					end := start + 1
					for ; end < len(content) && content[end] != '"'; end++ {
					}
					value := content[start+1 : end]
					if value != "" && value != "YOUR_" && !strings.HasPrefix(value, "PLACEHOLDER") && !strings.HasPrefix(value, "example") {
						fmt.Println("WARNING: Sensitive data detected in config file. Do NOT commit config.json with real API keys or tokens to version control!")
						fmt.Println("Use environment variables instead (see .env file).")
						return
					}
				}
			}
		}
	}
}

func checkFilePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		fmt.Printf("WARNING: Config file %s has overly permissive permissions (%o). Consider restricting to 0600.\n", path, mode)
	}
}

func checkWorkspaceIsolation(workspace string) error {
	info, err := os.Stat(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace is not a directory: %s", workspace)
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		fmt.Printf("WARNING: Workspace directory %s has overly permissive permissions (%o). Consider restricting to 0700.\n", workspace, mode)
	}
	return nil
}
