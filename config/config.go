package config

import (
	"encoding/json"
	"errors"
	"os"

	"muzzammil.xyz/jsonc"
)

type Settings struct {
	BotToken    string `json:"bot_token"`
	EnableDebug bool   `json:"enable_debug"`
	DBFile      string `json:"db_file"`
}

var _ json.Unmarshaler = (*Settings)(nil)

func (s *Settings) UnmarshalJSON(data []byte) error {
	type Alias Settings
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if s.BotToken == "" {
		return errors.New("bot_token is required")
	}

	if s.DBFile == "" {
		return errors.New("db_file is required")
	}

	return nil
}

func LoadSettings(filePath string) (Settings, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Settings{}, err
	}

	var config Settings
	if err := jsonc.Unmarshal(data, &config); err != nil {
		return Settings{}, err
	}
	return config, nil
}
