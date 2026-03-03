package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	DayBoundaryHour int                `json:"day_boundary_hour"`
	TokenCategories map[string]string  `json:"token_categories"`
	TokenWeights    map[string]float64 `json:"token_weights"`
	CalendarMap     map[string]string  `json:"calendar_keyword_categories"`
	WorkdayStart    int                `json:"workday_start"`
	WorkdayEnd      int                `json:"workday_end"`
	PrivacyRedact   bool               `json:"privacy_redact_titles"`
}

func Default() Config {
	return Config{
		DayBoundaryHour: 4,
		WorkdayStart:    9,
		WorkdayEnd:      18,
		PrivacyRedact:   false,
		TokenCategories: map[string]string{
			"😴": "sleep", "💤": "sleep", "zzz": "sleep",
			"🌅": "wake",
			"🚿": "hygiene", "🪥": "hygiene",
			"🍽️": "nutrition", "☕": "nutrition",
			"🏋️": "exercise", "🧗": "exercise", "🏃": "exercise",
			"🧠": "deepwork", "📚": "deepwork", "💻": "deepwork",
			"🎮":     "gaming",
			"📱":     "doomscroll",
			"🧑‍🤝‍🧑": "social",
			"😡":     "stress", "🤯": "stress",
			"🕌": "spiritual", "🚆": "travel",
		},
		TokenWeights: map[string]float64{"📱": 1.3, "🧠": 1.2, "🏃": 1.1},
		CalendarMap:  map[string]string{"meeting": "meeting", "standup": "meeting", "focus": "deepwork", "gym": "exercise", "commute": "travel"},
	}
}

func Save(path string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func Load(path string) (Config, error) {
	c := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}
