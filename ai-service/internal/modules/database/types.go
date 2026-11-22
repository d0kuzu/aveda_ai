package database

type Assistant struct {
	Id               string `json:"id,omitempty"`
	Name             string `json:"name,omitempty"`
	ApiToken         string `json:"api_token,omitempty"`
	UserId           string `json:"user_id,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	Configuration    string `json:"configuration,omitempty"`
	TelegramBotToken string `json:"telegram_bot_token,omitempty"`
	Type             string `json:"type,omitempty"`
}

type Chat struct {
	Id           string `json:"id,omitempty"`
	AssistantId  string `json:"assistant_id,omitempty"`
	CustomerId   string `json:"customer_id,omitempty"`
	Platform     string `json:"platform,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	MessageCount int32  `json:"message_count,omitempty"`
	IsEnd        bool   `json:"is_end,omitempty"`
}

type Message struct {
	Id        string `json:"id,omitempty"`
	ChatId    string `json:"chat_id,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	Platform  string `json:"platform,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type TwilioConfig struct {
	AssistantId  string
	UserId       string
	TwilioNumber string
	AccountSid   string
	AuthToken    string
}

type PeriodMetrics struct {
	StartedChats   int32
	CompletedChats int32
}

type PeriodChats struct {
	Chats      []*Chat
	TotalCount int64
}

type DailyCount struct {
	Date  string
	Count int32
}

type WeeklyChatsStarted struct {
	Days []*DailyCount
}
