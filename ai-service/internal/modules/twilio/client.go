package twilio

import (
	"github.com/twilio/twilio-go"
)

type Client struct {
}

func InitClient() *Client {
	return &Client{}
}

func (c *Client) GetRestClient(accountSID, authToken string) *twilio.RestClient {
	return twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
}
