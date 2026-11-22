package llm

import (
	"diaxel/internal/modules/database"

	"github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	db     *database.Store
	model  string
}

func InitClient(apiKey string, dbClient *database.Store, model string) *Client {
	if model == "" {
		model = "gpt-4o"
	}
	return &Client{client: openai.NewClient(apiKey), db: dbClient, model: model}
}
