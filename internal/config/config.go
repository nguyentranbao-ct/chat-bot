package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Database DatabaseConfig `envPrefix:"DATABASE_"`
	ChatAPI  ChatAPIConfig  `envPrefix:"CHAT_API_"`
	LLM      LLMConfig      `envPrefix:"LLM_"`
	Kafka    KafkaConfig    `envPrefix:"KAFKA_"`
}

type ServerConfig struct {
	Port string `env:"PORT" envDefault:"8080"`
	Host string `env:"HOST" envDefault:"0.0.0.0"`
}

type DatabaseConfig struct {
	Hosts    []string `env:"HOSTS" envDefault:"localhost:27017"`
	Username string   `env:"USERNAME" envDefault:""`
	Password string   `env:"PASSWORD" envDefault:""`
	Database string   `env:"DATABASE" envDefault:"chat-bot"`
	AuthDB   string   `env:"AUTH_DB" envDefault:"admin"`
	Direct   bool     `env:"DIRECT" envDefault:"true"`
}

type ChatAPIConfig struct {
	BaseURL   string `env:"BASE_URL,required" envDefault:"https://chat-dev.cmco.io"`
	ProjectID string `env:"PROJECT_ID,required" envDefault:"16f38160-3afa-4707-b8cb-354d2cbf1590"`
	APIKey    string `env:"API_KEY,required"`
	Service   string `env:"SERVICE" envDefault:"chat-bot"`
}

type LLMConfig struct {
	OpenAIAPIKey    string `env:"OPENAI_API_KEY"`
	AnthropicAPIKey string `env:"ANTHROPIC_API_KEY"`
	GoogleAIAPIKey  string `env:"GOOGLE_AI_API_KEY"`
}

type KafkaConfig struct {
	Enabled   bool     `env:"ENABLED" envDefault:"false"`
	Brokers   []string `env:"BROKERS" envDefault:"kafka-08.ct.dev:9092"`
	Topic     string   `env:"TOPIC" envDefault:"chat.event.messages"`
	GroupID   string   `env:"GROUP_ID" envDefault:"chat-bot-consumers"`
	Whitelist []string `env:"SELLER_WHITELIST" envDefault:"11198316"`
}

func Load() (*Config, error) {
	cfg := new(Config)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
