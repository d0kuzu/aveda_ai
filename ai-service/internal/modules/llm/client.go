package llm

import (
	"diaxel/internal/grpc/db"

	"github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	db     *db.Client
	model  string
}

func InitClient(apiKey string, dbClient *db.Client, model string) *Client {
	if model == "" {
		model = "gpt-4o"
	}
	return &Client{client: openai.NewClient(apiKey), db: dbClient, model: model}
}
