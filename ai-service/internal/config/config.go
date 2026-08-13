package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Settings struct {
	HTTPPort    string
	GRPCAddress string

	OpenaiApiKey string

	TwilioAccountSID string
	TwilioAuthToken  string

	TelegramWebhookSecret string
	WebhookBaseURL        string
	BaseURL               string

	TokenPrefix string
	TokenLength int

	CampusLoginAPI string

	MaxSlotCapacity      int
	SlotDurationMinutes  int
	WorkSchedule         string
	AvedaCalendarSecret  string
}

func LoadConfig() (*Settings, error) {
	godotenv.Load(".env")

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	grpcAddress := os.Getenv("GRPC_ADDRESS")
	if grpcAddress == "" {
		grpcAddress = "localhost:50051"
	}

	workSchedule := os.Getenv("WORK_SCHEDULE")
	if workSchedule == "" {
		workSchedule = "Tue=09:00-17:00,Wed=11:00-19:00,Thu=11:00-19:00,Fri=09:00-17:00,Sat=09:00-17:00"
	}

	return &Settings{
		HTTPPort:    httpPort,
		GRPCAddress: grpcAddress,

		OpenaiApiKey: os.Getenv("OPENAI_API_KEY"),

		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),

		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		WebhookBaseURL:        os.Getenv("WEBHOOK_BASE_URL"),
		BaseURL:               os.Getenv("BASE_URL"),

		TokenPrefix: os.Getenv("TOKEN_PREFIX"),
		TokenLength: getEnvAsInt("TOKEN_LENGTH", 32),

		CampusLoginAPI: os.Getenv("CAMPUSLOGIN_API"),

		MaxSlotCapacity:     getEnvAsInt("MAX_SLOT_CAPACITY", 10),
		SlotDurationMinutes: getEnvAsInt("SLOT_DURATION_MINUTES", 60),
		WorkSchedule:        workSchedule,
		AvedaCalendarSecret: os.Getenv("AVEDA_CALENDAR_SECRET"),
	}, nil
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultVal
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultVal
	}

	return value
}

