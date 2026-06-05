package db

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dbpb "diaxel/proto/db"
)

type Client struct {
	conn *grpc.ClientConn
	DB   dbpb.DatabaseServiceClient
}

func New(address string) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания gRPC клиента: %w", err)
	}

	dbClient := dbpb.NewDatabaseServiceClient(conn)

	return &Client{
		conn: conn,
		DB:   dbClient,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreateAssistant(name, token, userId, telegramBotToken, assistantType string) (*dbpb.AssistantResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.CreateAssistantRequest{
		Name:             name,
		ApiToken:         token,
		UserId:           userId,
		TelegramBotToken: telegramBotToken,
		Type:             assistantType,
	}

	resp, err := c.DB.CreateAssistant(ctx, req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetAssistant(id string) (*dbpb.AssistantResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetAssistantRequest{
		Id: id,
	}

	resp, err := c.DB.GetAssistant(ctx, req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) UpdateAssistant(id, name, configuration, apiToken, telegramBotToken, assistantType string) (*dbpb.AssistantResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.UpdateAssistantRequest{
		Id:               id,
		Name:             name,
		Configuration:    configuration,
		ApiToken:         apiToken,
		TelegramBotToken: telegramBotToken,
		Type:             assistantType,
	}

	return c.DB.UpdateAssistant(ctx, req)
}

func (c *Client) GetAssistantByAPIToken(apiToken string) (*dbpb.AssistantResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetAssistantByAPITokenRequest{
		ApiToken: apiToken,
	}

	resp, err := c.DB.GetAssistantByAPIToken(ctx, req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetAssistantsByUserID(userID string) ([]*dbpb.AssistantResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetAssistantsByUserIDRequest{
		UserId: userID,
	}

	resp, err := c.DB.GetAssistantsByUserID(ctx, req)

	if err != nil {
		return nil, err
	}

	if resp.Assistants == nil {
		return []*dbpb.AssistantResponse{}, nil
	}

	return resp.Assistants, nil
}

func (c *Client) CreateChat(assistantID, customerID, platform string) (*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.CreateChatRequest{
		AssistantId: assistantID,
		CustomerId:  customerID,
		Platform:    platform,
	}

	return c.DB.CreateChat(ctx, req)
}

func (c *Client) GetChat(chatID string) (*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatRequest{
		Id: chatID,
	}

	return c.DB.GetChat(ctx, req)
}

func (c *Client) SaveMessage(chatID, role, content, platform string) (*dbpb.MessageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.SaveMessageRequest{
		ChatId:   chatID,
		Role:     role,
		Content:  content,
		Platform: platform,
	}

	return c.DB.SaveMessage(ctx, req)
}

func (c *Client) GetAllChatMessages(chatID string) ([]*dbpb.MessageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetAllChatMessagesRequest{
		ChatId: chatID,
	}

	resp, err := c.DB.GetAllChatMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Messages == nil {
		return []*dbpb.MessageResponse{}, nil
	}

	return resp.Messages, nil
}

func (c *Client) GetLatestChatByCustomer(assistantID, customerID string) (*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetLatestChatByCustomerRequest{
		AssistantId: assistantID,
		CustomerId:  customerID,
	}

	return c.DB.GetLatestChatByCustomer(ctx, req)
}

func (c *Client) GetChatMessages(chatID string, limit, offset int32) ([]*dbpb.MessageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatMessagesRequest{
		ChatId: chatID,
		Limit:  limit,
		Offset: offset,
	}

	resp, err := c.DB.GetChatMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Messages == nil {
		return []*dbpb.MessageResponse{}, nil
	}

	return resp.Messages, nil
}

func (c *Client) GetChatPage(assistantID string, page, chatsPerPage int32) ([]*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatPageRequest{
		AssistantId:  assistantID,
		Page:         page,
		ChatsPerPage: chatsPerPage,
	}

	resp, err := c.DB.GetChatPage(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Chats == nil {
		return []*dbpb.ChatResponse{}, nil
	}

	return resp.Chats, nil
}

func (c *Client) GetChatPagesCountByUserID(assistantIDs []string, chatsPerPage int32) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatPagesCountByUserIDRequest{
		AssistantIds: assistantIDs,
		ChatsPerPage: chatsPerPage,
	}

	resp, err := c.DB.GetChatPagesCountByUserID(ctx, req)
	if err != nil {
		return 0, err
	}

	return resp.PagesCount, nil
}

func (c *Client) GetChatPageByUserID(assistantIDs []string, page, chatsPerPage int32) ([]*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatPageByUserIDRequest{
		AssistantIds: assistantIDs,
		Page:         page,
		ChatsPerPage: chatsPerPage,
	}

	resp, err := c.DB.GetChatPageByUserID(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Chats == nil {
		return []*dbpb.ChatResponse{}, nil
	}

	return resp.Chats, nil
}

func (c *Client) GetChatPagesCount(assistantID string, chatsPerPage int32) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatPagesCountRequest{
		AssistantId:  assistantID,
		ChatsPerPage: chatsPerPage,
	}

	resp, err := c.DB.GetChatPagesCount(ctx, req)
	if err != nil {
		return 0, err
	}

	return resp.PagesCount, nil
}

func (c *Client) SearchChatsByCustomer(assistantIDs []string, search string) ([]*dbpb.ChatResponse, int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.SearchChatsByCustomerRequest{
		AssistantIds: assistantIDs,
		Search:       search,
	}

	resp, err := c.DB.SearchChatsByCustomer(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	if resp.Chats == nil {
		return []*dbpb.ChatResponse{}, resp.TotalCount, nil
	}

	return resp.Chats, resp.TotalCount, nil
}

func (c *Client) GetTwilioConfig(assistantID string) (*dbpb.TwilioConfigResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetTwilioConfigRequest{
		AssistantId: assistantID,
	}

	return c.DB.GetTwilioConfig(ctx, req)
}

func (c *Client) SaveTwilioConfig(assistantID, userID, twilioNumber, accountSID, authToken string) (*dbpb.TwilioConfigResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.SaveTwilioConfigRequest{
		AssistantId:  assistantID,
		UserId:       userID,
		TwilioNumber: twilioNumber,
		AccountSid:   accountSID,
		AuthToken:    authToken,
	}

	return c.DB.SaveTwilioConfig(ctx, req)
}

func (c *Client) GetCampusloginByUserId(userID string) (*dbpb.CampusloginResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.CampusloginRequest{
		UserId: userID,
	}

	resp, err := c.DB.GetCampusloginByUserId(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) UpsertCampuslogin(userID string, contactID int, programID int, isGrade11OrLower bool, isInternationalStudent bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.UpsertCampusloginRequest{
		UserId:                 userID,
		ContactId:              int32(contactID),
		ProgramId:              int32(programID),
		IsGrade11OrLower:       isGrade11OrLower,
		IsInternationalStudent: isInternationalStudent,
	}

	_, err := c.DB.UpsertCampuslogin(ctx, req)
	return err
}

func (c *Client) DeleteAllChatsAndMessages() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &dbpb.DeleteAllChatsAndMessagesRequest{}
	_, err := c.DB.DeleteAllChatsAndMessages(ctx, req)
	return err
}

func (c *Client) DeleteChatAndMessages(chatID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &dbpb.DeleteChatAndMessagesRequest{
		ChatId: chatID,
	}
	_, err := c.DB.DeleteChatAndMessages(ctx, req)
	return err
}

func (c *Client) UpdateChatIsEnd(chatID string, isEnd bool) (*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.UpdateChatIsEndRequest{
		Id:    chatID,
		IsEnd: isEnd,
	}

	return c.DB.UpdateChatIsEnd(ctx, req)
}

func (c *Client) UpdateChatIsReviewed(chatID string, isReviewed bool) (*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.UpdateChatIsReviewedRequest{
		Id:         chatID,
		IsReviewed: isReviewed,
	}

	return c.DB.UpdateChatIsReviewed(ctx, req)
}

func (c *Client) GetUnreviewedActiveChats() ([]*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetUnreviewedActiveChatsRequest{}

	resp, err := c.DB.GetUnreviewedActiveChats(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Chats, nil
}

func (c *Client) GetChatsForFollowup() ([]*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetChatsForFollowupRequest{}

	resp, err := c.DB.GetChatsForFollowup(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Chats, nil
}

func (c *Client) UpdateChatFollowupStage(chatID string, stage int32) (*dbpb.ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.UpdateChatFollowupStageRequest{
		Id:    chatID,
		Stage: stage,
	}

	return c.DB.UpdateChatFollowupStage(ctx, req)
}

func (c *Client) SetCampusloginFlags(userID string, isGrade11OrLower bool, isInternationalStudent bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.SetCampusloginFlagsRequest{
		UserId:                 userID,
		IsGrade11OrLower:       isGrade11OrLower,
		IsInternationalStudent: isInternationalStudent,
	}

	_, err := c.DB.SetCampusloginFlags(ctx, req)
	return err
}

func (c *Client) GetPeriodMetrics(assistantID, startTime, endTime string) (*dbpb.GetPeriodMetricsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetPeriodMetricsRequest{
		AssistantId: assistantID,
		StartTime:   startTime,
		EndTime:     endTime,
	}

	return c.DB.GetPeriodMetrics(ctx, req)
}

func (c *Client) GetWeeklyChatsStarted(assistantID, startTime string) (*dbpb.GetWeeklyChatsStartedResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &dbpb.GetWeeklyChatsStartedRequest{
		AssistantId: assistantID,
		StartTime:   startTime,
	}

	return c.DB.GetWeeklyChatsStarted(ctx, req)
}
