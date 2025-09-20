package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Database DatabaseConfig `envPrefix:"DATABASE_"`
	ChatAPI  ChatAPIConfig  `envPrefix:"CHAT_API_"`
	LLM      LLMConfig      `envPrefix:"LLM_"`
}

type ServerConfig struct {
	Port string `env:"PORT" envDefault:"8080"`
	Host string `env:"HOST" envDefault:"0.0.0.0"`
}

type DatabaseConfig struct {
	URI      string `env:"URI" envDefault:"mongodb://localhost:27017"`
	Database string `env:"DATABASE" envDefault:"chatbot"`
}

type ChatAPIConfig struct {
	BaseURL     string `env:"BASE_URL,required"`
	ProjectUUID string `env:"PROJECT_UUID,required"`
	Service     string `env:"SERVICE" envDefault:"chat-bot"`
}

type LLMConfig struct {
	OpenAIAPIKey    string `env:"OPENAI_API_KEY"`
	AnthropicAPIKey string `env:"ANTHROPIC_API_KEY"`
	GoogleAIAPIKey  string `env:"GOOGLE_AI_API_KEY"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}