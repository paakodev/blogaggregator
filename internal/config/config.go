package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (*Config, error) {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	reader, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	config := &Config{}
	if err := json.NewDecoder(reader).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) SetUser(userName string) error {
	c.CurrentUserName = userName
	return c.write()
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if homeDir == "" {
		return "", fmt.Errorf("could not find home directory, got empty string")
	}
	return filepath.Join(homeDir, configFileName), nil
}

func (c *Config) write() error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFilePath, jsonData, 0644)
}
