package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type config struct {
	AutoColorEnabled bool    `json:"auto_color_enabled"`
	DayTemp          int     `json:"day_temp"`
	NightTemp        int     `json:"night_temp"`
	ManualTemp       int     `json:"manual_temp"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
}

var cfg config

func configPath() string {
	return filepath.Join(dataDir, "config.json")
}

func loadConfig() {
	data, err := os.ReadFile(configPath())
	if err != nil {
		log.Printf("config: no config file, using defaults")
		applyConfigDefaults()
		return
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("config: parse error: %v, using defaults", err)
	}
	applyConfigDefaults()
	log.Printf("config: loaded (auto_color=%v day=%dK night=%dK lat=%.2f lon=%.2f)",
		cfg.AutoColorEnabled, cfg.DayTemp, cfg.NightTemp, cfg.Latitude, cfg.Longitude)
}

func applyConfigDefaults() {
	if cfg.DayTemp == 0 {
		cfg.DayTemp = 6500
	}
	if cfg.NightTemp == 0 {
		cfg.NightTemp = 3500
	}
	if cfg.ManualTemp == 0 {
		cfg.ManualTemp = 6500
	}
}

func saveConfig() {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("config: marshal error: %v", err)
		return
	}
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("config: write error: %v", err)
		return
	}
	if err := os.Rename(tmp, configPath()); err != nil {
		log.Printf("config: rename error: %v", err)
		return
	}
}
