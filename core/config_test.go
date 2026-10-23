package core

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaultsWhenMissing(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir())
	if got, want := LoadConfig(), DefaultConfig(); got != want {
		t.Errorf("missing file should yield defaults: got %+v", got)
	}
}

func TestLoadConfigPartialOverride(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir())
	writeConfig(t, `{"access_half_life_days": 7, "text_weight_long": 0.9}`)

	cfg := LoadConfig()
	if cfg.AccessHalfLifeDays != 7 {
		t.Errorf("access_half_life_days = %v, want 7", cfg.AccessHalfLifeDays)
	}
	if cfg.TextWeightLong != 0.9 {
		t.Errorf("text_weight_long = %v, want 0.9", cfg.TextWeightLong)
	}
	// Fields absent from the file keep their defaults.
	def := DefaultConfig()
	if cfg.TimeDecayDays != def.TimeDecayDays || cfg.FrequencySaturation != def.FrequencySaturation {
		t.Errorf("unset fields should keep defaults: got %+v", cfg)
	}
}

func TestLoadConfigSanitizesInvalidValues(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir())
	writeConfig(t, `{"time_decay_days": -5, "frequency_saturation": 0, "text_weight_short": 1.5, "fallback_damping": -1}`)

	cfg := LoadConfig()
	def := DefaultConfig()
	if cfg.TimeDecayDays != def.TimeDecayDays {
		t.Errorf("non-positive time_decay_days should fall back to default, got %v", cfg.TimeDecayDays)
	}
	if cfg.FrequencySaturation != def.FrequencySaturation {
		t.Errorf("non-positive frequency_saturation should fall back to default, got %v", cfg.FrequencySaturation)
	}
	if cfg.TextWeightShort != 1.0 {
		t.Errorf("text_weight_short should clamp to 1.0, got %v", cfg.TextWeightShort)
	}
	if cfg.FallbackDamping != 0.0 {
		t.Errorf("fallback_damping should clamp to 0.0, got %v", cfg.FallbackDamping)
	}
}

func TestLoadConfigCorruptFile(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir())
	writeConfig(t, `{not json`)
	if got, want := LoadConfig(), DefaultConfig(); got != want {
		t.Errorf("corrupt file should yield defaults: got %+v", got)
	}
}

func TestWriteDefaultConfigFile(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir())

	path, err := WriteDefaultConfigFile()
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if got, want := LoadConfig(), DefaultConfig(); got != want {
		t.Errorf("scaffolded file should round-trip the defaults: got %+v", got)
	}

	// Refuses to overwrite.
	if _, err := WriteDefaultConfigFile(); err == nil {
		t.Errorf("second write to %s should fail", path)
	}
}

// TestConfigAffectsRanking verifies end to end that a config file changes the
// blend: with text weight forced to 1.0 for short queries, the better text
// match must win even against a heavily-used directory (the opposite of the
// default-behavior assertion in TestShortQueryUsageBreaksWeakTextTie).
func TestConfigAffectsRanking(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir())
	writeConfig(t, `{"text_weight_short": 1.0}`)

	now := time.Now()
	old := now.Add(-30 * 24 * time.Hour)
	dirs := []Directory{
		{Name: "2025-01-01-redis-lab", Path: "/x/lab", ModifiedTime: old},
		{Name: "2025-01-01-redis-cache", Path: "/x/cache", ModifiedTime: old,
			AccessCount: 25, LastAccess: now},
	}
	scored := FilterAndScoreDirectories(dirs, "re")
	SortDirectoriesByScore(scored)
	if scored[0].Name != "2025-01-01-redis-lab" {
		t.Errorf("with text_weight_short=1.0 the better text match should win, got %v", names(scored))
	}
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(GetConfigPath(), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
