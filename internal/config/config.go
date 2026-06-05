package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                  string
	LogLevel              string
	CursorFilePath        string
	CursorRewindSeconds   int
	HydrationWorkers      int
	ClassificationWorkers int
	GrazeFeedURI          string
	ContrailsWSURL        string
	SlingshotURL          string
	LLMEndpoint           string
	LLMModel              string
	LLMAPIKey             string
	LLMTemperature        float64
	OzoneEndpoint         string
	LabelerDID            string
	OzoneAdminToken       string
	DryRun                bool
	LLMSystemPrompt       string
}

func Load() (*Config, error) {
	systemPrompt, err := loadSystemPrompt(getEnv("LLM_SYSTEM_PROMPT", ""), getEnv("LLM_SYSTEM_PROMPT_PATH", ""))
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:                  getEnv("PORT", "8081"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		CursorFilePath:        getEnv("CURSOR_FILE_PATH", "./data/cursor.json"),
		CursorRewindSeconds:   getEnvInt("CURSOR_REWIND_SECONDS", 10),
		HydrationWorkers:      getEnvInt("HYDRATION_WORKERS", 10),
		ClassificationWorkers: getEnvInt("CLASSIFICATION_WORKERS", 4),
		GrazeFeedURI:          getEnv("GRAZE_FEED_URI", ""),
		ContrailsWSURL:        getEnv("CONTRAILS_WS_URL", "wss://api.graze.social/app/contrail"),
		SlingshotURL:          getEnv("SLINGSHOT_URL", "https://slingshot.microcosm.blue"),
		LLMEndpoint:           getEnv("LLM_ENDPOINT", "http://localhost:8080/v1/"),
		LLMModel:              getEnv("LLM_MODEL", "google/gemma-4-e2b-gguf"),
		LLMAPIKey:             getEnv("LLM_API_KEY", ""),
		LLMTemperature:        getEnvFloat("LLM_TEMPERATURE", 0.0),
		OzoneEndpoint:         getEnv("OZONE_ENDPOINT", "http://localhost:3000"),
		LabelerDID:            getEnv("LABELER_DID", ""),
		OzoneAdminToken:       getEnv("OZONE_ADMIN_TOKEN", ""),
		DryRun:                getEnvBool("DRY_RUN", false),
		LLMSystemPrompt:       systemPrompt,
	}, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}

func getEnvFloat(key string, fallback float64) float64 {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return fallback
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return fallback
	}
	return val
}

func getEnvBool(key string, fallback bool) bool {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return fallback
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return fallback
	}
	return val
}

func loadSystemPrompt(promptEnv, promptPathEnv string) (string, error) {
	if promptEnv != "" {
		return promptEnv, nil
	}
	if promptPathEnv != "" {
		content, err := os.ReadFile(promptPathEnv)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return "", nil
}
