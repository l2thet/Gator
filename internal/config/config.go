package config

import (
	"encoding/json"
	"log"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		log.Fatalf("Error getting config file path: %v", err)
		return Config{}, err
	}

	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling config file: %v", err)
		return Config{}, err
	}

	return config, nil
}

func (cfg *Config) SetUser(name string) {
	cfg.CurrentUserName = name

	err := write(*cfg)
	if err != nil {
		log.Fatalf("Error writing config file: %v", err)
	}
}

func getConfigFilePath() (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path + "/" + configFileName, nil
}

func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	file, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, file, os.FileMode(0644))
	if err != nil {
		return err
	}

	return nil
}
