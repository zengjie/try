package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the user-tunable ranking parameters. Every field has a safe
// default; a missing or partial config file simply keeps the defaults, in line
// with the zero-configuration design goal.
type Config struct {
	// TimeDecayDays is the half-life, in days, of the directory-mtime recency
	// signal used for directories never opened through try.
	TimeDecayDays float64 `json:"time_decay_days"`
	// AccessHalfLifeDays is the half-life, in days, of the frecency recency
	// signal (how recently the directory was opened through try).
	AccessHalfLifeDays float64 `json:"access_half_life_days"`
	// FrequencySaturation is the K in count/(count+K): the number of visits at
	// which the frequency component reaches half of its maximum.
	FrequencySaturation float64 `json:"frequency_saturation"`
	// FallbackDamping scales the mtime-recency fallback for directories with no
	// access history, in [0,1]. Lower values favor directories you actually open.
	FallbackDamping float64 `json:"fallback_damping"`
	// TextWeightShort/Medium/Long control how much the text-match signal counts
	// versus the usage signal for 1-2, 3-4, and 5+ character queries, in [0,1].
	TextWeightShort  float64 `json:"text_weight_short"`
	TextWeightMedium float64 `json:"text_weight_medium"`
	TextWeightLong   float64 `json:"text_weight_long"`
}

// DefaultConfig returns the built-in ranking parameters.
func DefaultConfig() Config {
	return Config{
		TimeDecayDays:       30,
		AccessHalfLifeDays:  14,
		FrequencySaturation: 3,
		FallbackDamping:     0.9,
		TextWeightShort:     0.45,
		TextWeightMedium:    0.65,
		TextWeightLong:      0.80,
	}
}

// GetConfigPath returns where the config file lives (next to history.json).
func GetConfigPath() string {
	return filepath.Join(getDataDir(), "config.json")
}

// LoadConfig reads the config file and merges it over the defaults: fields
// missing from the file keep their default value and invalid values are
// corrected. It never fails the caller — a missing or corrupt file yields the
// defaults so ranking always works.
func LoadConfig() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(GetConfigPath())
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}
	return sanitizeConfig(cfg)
}

// sanitizeConfig fixes out-of-range values field by field: scales that must be
// positive fall back to their default, weights are clamped into [0,1].
func sanitizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if cfg.TimeDecayDays <= 0 {
		cfg.TimeDecayDays = def.TimeDecayDays
	}
	if cfg.AccessHalfLifeDays <= 0 {
		cfg.AccessHalfLifeDays = def.AccessHalfLifeDays
	}
	if cfg.FrequencySaturation <= 0 {
		cfg.FrequencySaturation = def.FrequencySaturation
	}
	cfg.FallbackDamping = clamp01(cfg.FallbackDamping)
	cfg.TextWeightShort = clamp01(cfg.TextWeightShort)
	cfg.TextWeightMedium = clamp01(cfg.TextWeightMedium)
	cfg.TextWeightLong = clamp01(cfg.TextWeightLong)
	return cfg
}

// WriteDefaultConfigFile creates the config file populated with the defaults so
// the user can edit it. It refuses to overwrite an existing file.
func WriteDefaultConfigFile() (string, error) {
	path := GetConfigPath()
	if _, err := os.Stat(path); err == nil {
		return path, fmt.Errorf("config file already exists: %s", path)
	}
	if err := os.MkdirAll(getDataDir(), 0755); err != nil {
		return path, err
	}
	data, err := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err != nil {
		return path, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return path, err
	}
	return path, nil
}
