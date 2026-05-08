package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Addr               string `json:"addr"`
	Data               string `json:"data"`
	DB                 string `json:"db"`
	PublicURL          string `json:"public_url"`
	GitHubClientID     string `json:"github_client_id"`
	GitHubClientSecret string `json:"github_client_secret"`
}

func Defaults() Config {
	return Config{Addr: ":8080", Data: "./data"}
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	if env := os.Getenv("CLICKCLACK_ADDR"); env != "" {
		cfg.Addr = env
	}
	if env := os.Getenv("CLICKCLACK_DATA"); env != "" {
		cfg.Data = env
	}
	if env := os.Getenv("CLICKCLACK_DB"); env != "" {
		cfg.DB = env
	}
	if env := os.Getenv("CLICKCLACK_PUBLIC_URL"); env != "" {
		cfg.PublicURL = env
	}
	if env := os.Getenv("CLICKCLACK_GITHUB_CLIENT_ID"); env != "" {
		cfg.GitHubClientID = env
	}
	if env := os.Getenv("CLICKCLACK_GITHUB_CLIENT_SECRET"); env != "" {
		cfg.GitHubClientSecret = env
	}
	if path == "" {
		return cfg, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.Data == "" {
		cfg.Data = "./data"
	}
	return cfg, nil
}
