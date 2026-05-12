package twilio

import (
	"context"
	"fmt"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

func (c *Client) SendMessage(ctx context.Context, accountSID, authToken, from, to, message string) error {
	client := c.GetRestClient(accountSID, authToken)

	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(from)
	params.SetBody(message)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		return err
	}

	fmt.Println("Message SID:", *resp.Sid)
	return nil
}
