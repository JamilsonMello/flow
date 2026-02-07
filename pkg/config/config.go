package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DB     DBConfig     `yaml:"db"`
	Server ServerConfig `yaml:"server"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	cfg := &Config{}
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %v", err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8585
	}
	if cfg.DB.Host == "" {
		cfg.DB.Host = "127.0.0.1"
	}
	if cfg.DB.Port == 0 {
		cfg.DB.Port = 5432
	}

	return cfg, nil
}

func (c *Config) GetConnString() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s port=%d",
		c.DB.User, c.DB.Password, c.DB.Name, c.DB.Host, c.DB.Port)
}
