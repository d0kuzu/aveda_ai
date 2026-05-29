package server

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"diaxel_zerde/database-service/models"
	"diaxel_zerde/database-service/proto"
	"diaxel_zerde/database-service/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DatabaseServer struct {
	proto.UnimplementedDatabaseServiceServer
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	chatRepo         repository.ChatRepository
	messageRepo      repository.MessageRepository
	assistantRepo    repository.AssistantRepository
	twilioRepo       repository.TwilioRepository
	campusloginRepo  *repository.CampusloginRepository
}

func NewDatabaseServer(userRepo repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository, chatRepo repository.ChatRepository, messageRepo repository.MessageRepository, assistantRepo repository.AssistantRepository, twilioRepo repository.TwilioRepository, campusloginRepo *repository.CampusloginRepository) *DatabaseServer {
	return &DatabaseServer{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		chatRepo:         chatRepo,
		messageRepo:      messageRepo,
		assistantRepo:    assistantRepo,
		twilioRepo:       twilioRepo,
		campusloginRepo:  campusloginRepo,
	}
}

func (s *DatabaseServer) CreateAssistant(ctx context.Context, req *proto.CreateAssistantRequest) (*proto.AssistantResponse, error) {
	assistant, err := s.assistantRepo.CreateAssistant(ctx, req.Name, req.ApiToken, req.UserId, req.TelegramBotToken, req.Type)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create assistant: %v", err)
	}

	return &proto.AssistantResponse{
		Id:               assistant.ID,
		Name:             assistant.Name,
		ApiToken:         assistant.APIToken,
		UserId:           assistant.UserID,
		CreatedAt:        assistant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        assistant.UpdatedAt.Format(time.RFC3339),
		Configuration:    assistant.Configuration,
		TelegramBotToken: assistant.TelegramBotToken,
		Type:             assistant.Type,
	}, nil
}

func (s *DatabaseServer) CreateUser(ctx context.Context, req *proto.CreateUserRequest) (*proto.UserResponse, error) {
	user, err := s.userRepo.CreateUser(ctx, req.Email, req.PasswordHash, req.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	var email, passwordHash, role string
	if user.Email != nil {
		email = *user.Email
	}
	if user.PasswordHash != nil {
		passwordHash = *user.PasswordHash
	}
	if user.Role != nil {
		role = *user.Role
	}

	return &proto.UserResponse{
		Id:           user.ID,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) GetUser(ctx context.Context, req *proto.GetUserRequest) (*proto.UserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	var email, passwordHash, role string
	if user.Email != nil {
		email = *user.Email
	}
	if user.PasswordHash != nil {
		passwordHash = *user.PasswordHash
	}
	if user.Role != nil {
		role = *user.Role
	}

	return &proto.UserResponse{
		Id:           user.ID,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) GetUserByEmail(ctx context.Context, req *proto.GetUserByEmailRequest) (*proto.UserResponse, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	var email, passwordHash, role string
	if user.Email != nil {
		email = *user.Email
	}
	if user.PasswordHash != nil {
		passwordHash = *user.PasswordHash
	}
	if user.Role != nil {
		role = *user.Role
	}

	return &proto.UserResponse{
		Id:           user.ID,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) UpdateUser(ctx context.Context, req *proto.UpdateUserRequest) (*proto.UserResponse, error) {
	user, err := s.userRepo.UpdateUser(ctx, req.Id, req.Email, req.PasswordHash, req.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	var email, passwordHash, role string
	if user.Email != nil {
		email = *user.Email
	}
	if user.PasswordHash != nil {
		passwordHash = *user.PasswordHash
	}
	if user.Role != nil {
		role = *user.Role
	}

	return &proto.UserResponse{
		Id:           user.ID,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) DeleteUser(ctx context.Context, req *proto.DeleteUserRequest) (*proto.DeleteUserResponse, error) {
	err := s.userRepo.DeleteUser(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &proto.DeleteUserResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) SaveRefreshToken(ctx context.Context, req *proto.SaveRefreshTokenRequest) (*proto.SaveRefreshTokenResponse, error) {
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid expires_at format: %v", err)
	}

	err = s.refreshTokenRepo.SaveRefreshToken(ctx, req.UserId, req.TokenHash, expiresAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save refresh token: %v", err)
	}

	return &proto.SaveRefreshTokenResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) GetRefreshToken(ctx context.Context, req *proto.GetRefreshTokenRequest) (*proto.RefreshTokenResponse, error) {
	token, err := s.refreshTokenRepo.GetRefreshToken(ctx, req.TokenHash)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "refresh token not found: %v", err)
	}

	return &proto.RefreshTokenResponse{
		Id:        token.ID,
		UserId:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt.Format(time.RFC3339),
		CreatedAt: token.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) DeleteRefreshToken(ctx context.Context, req *proto.DeleteRefreshTokenRequest) (*proto.DeleteRefreshTokenResponse, error) {
	err := s.refreshTokenRepo.DeleteRefreshToken(ctx, req.TokenHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete refresh token: %v", err)
	}

	return &proto.DeleteRefreshTokenResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) GetAnalytics(ctx context.Context, req *proto.AnalyticsRequest) (*proto.AnalyticsResponse, error) {
	// TODO: Implement analytics logic
	return &proto.AnalyticsResponse{
		AssistantId:    req.AssistantId,
		TotalChats:     0,
		ActiveUsers:    0,
		EngagementRate: 0.0,
	}, nil
}

func (s *DatabaseServer) GetAnalyticsByAssistant(ctx context.Context, req *proto.AnalyticsByAssistantRequest) (*proto.AnalyticsResponse, error) {
	// TODO: Implement analytics by assistant logic
	return &proto.AnalyticsResponse{
		AssistantId:    req.AssistantId,
		TotalChats:     0,
		ActiveUsers:    0,
		EngagementRate: 0.0,
	}, nil
}

func (s *DatabaseServer) CreateChat(ctx context.Context, req *proto.CreateChatRequest) (*proto.ChatResponse, error) {
	// Verify that the assistant exists before creating a chat
	_, err := s.assistantRepo.GetAssistantByID(ctx, req.AssistantId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "assistant not found: %v", err)
	}

	chat, err := s.chatRepo.CreateChat(ctx, req.AssistantId, req.CustomerId, req.Platform)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chat: %v", err)
	}

	return &proto.ChatResponse{
		Id:          chat.ID,
		AssistantId: chat.AssistantID,
		CustomerId: func() string {
			if chat.CustomerID != nil {
				return *chat.CustomerID
			}
			return ""
		}(),
		Platform:  req.Platform,
		CreatedAt: chat.StartedAt.Format(time.RFC3339),
		UpdatedAt: chat.StartedAt.Format(time.RFC3339),
		IsEnd:     chat.IsEnd,
	}, nil
}

func (s *DatabaseServer) GetChat(ctx context.Context, req *proto.GetChatRequest) (*proto.ChatResponse, error) {
	chat, err := s.chatRepo.GetChatByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "chat not found: %v", err)
	}

	return &proto.ChatResponse{
		Id:          chat.ID,
		AssistantId: chat.AssistantID,
		CustomerId: func() string {
			if chat.CustomerID != nil {
				return *chat.CustomerID
			}
			return ""
		}(),
		CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
		MessageCount: chat.MessageCount,
	}, nil
}

func (s *DatabaseServer) GetChatsByUser(ctx context.Context, req *proto.GetChatsByUserRequest) (*proto.ChatsResponse, error) {
	chats, err := s.chatRepo.GetChatsByUserID(ctx, req.AssistantIds, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chats by user: %v", err)
	}

	var protoChats []*proto.ChatResponse
	for _, chat := range chats {
		customerId := ""
		if chat.CustomerID != nil {
			customerId = *chat.CustomerID
		}

		protoChats = append(protoChats, &proto.ChatResponse{
			Id:           chat.ID,
			AssistantId:  chat.AssistantID,
			CustomerId:   customerId,
			CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
			MessageCount: chat.MessageCount,
		})
	}

	return &proto.ChatsResponse{
		Chats: protoChats,
	}, nil
}

func (s *DatabaseServer) UpdateChat(ctx context.Context, req *proto.UpdateChatRequest) (*proto.ChatResponse, error) {
	chat, err := s.chatRepo.UpdateChat(ctx, req.Id, req.AssistantId, req.CustomerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update chat: %v", err)
	}

	return &proto.ChatResponse{
		Id:          chat.ID,
		AssistantId: chat.AssistantID,
		CustomerId: func() string {
			if chat.CustomerID != nil {
				return *chat.CustomerID
			}
			return ""
		}(),
		CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
		MessageCount: chat.MessageCount,
	}, nil
}

func (s *DatabaseServer) DeleteChat(ctx context.Context, req *proto.DeleteChatRequest) (*proto.DeleteChatResponse, error) {
	err := s.chatRepo.DeleteChat(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete chat: %v", err)
	}

	return &proto.DeleteChatResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) SaveMessage(ctx context.Context, req *proto.SaveMessageRequest) (*proto.MessageResponse, error) {
	message, err := s.messageRepo.SaveMessage(ctx, req.GetChatId(), req.GetRole(), req.GetContent(), req.GetPlatform())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save message: %v", err)
	}

	// Update message count for the chat
	if err := s.chatRepo.UpdateMessageCount(ctx, req.GetChatId()); err != nil {
		// Log error but don't fail the message save
		// TODO: Add proper logging
	}

	return &proto.MessageResponse{
		Id:        fmt.Sprintf("%d", message.ID),
		ChatId:    message.ChatID,
		Role:      message.Role,
		Content:   message.Content,
		Platform:  req.GetPlatform(),
		CreatedAt: message.Time.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) GetChatMessages(ctx context.Context, req *proto.GetChatMessagesRequest) (*proto.MessagesResponse, error) {
	messages, err := s.messageRepo.GetMessagesByChatID(ctx, req.GetChatId(), req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}

	var protoMessages []*proto.MessageResponse
	for _, msg := range messages {
		protoMessages = append(protoMessages, &proto.MessageResponse{
			Id:        fmt.Sprintf("%d", msg.ID),
			ChatId:    msg.ChatID,
			Role:      msg.Role,
			Content:   msg.Content,
			Platform:  "", // Platform not stored in message model
			CreatedAt: msg.Time.Format(time.RFC3339),
		})
	}

	return &proto.MessagesResponse{
		Messages: protoMessages,
	}, nil
}

func (s *DatabaseServer) GetAllChatMessages(ctx context.Context, req *proto.GetAllChatMessagesRequest) (*proto.MessagesResponse, error) {
	messages, err := s.messageRepo.GetAllChatMessages(ctx, req.ChatId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get all messages: %v", err)
	}

	var protoMessages []*proto.MessageResponse
	for _, msg := range messages {
		protoMessages = append(protoMessages, &proto.MessageResponse{
			Id:        fmt.Sprintf("%d", msg.ID),
			ChatId:    msg.ChatID,
			Role:      msg.Role,
			Content:   msg.Content,
			Platform:  "", // Platform not stored in message model
			CreatedAt: msg.Time.Format(time.RFC3339),
		})
	}

	return &proto.MessagesResponse{
		Messages: protoMessages,
	}, nil
}

func (s *DatabaseServer) UpdateMessage(ctx context.Context, req *proto.UpdateMessageRequest) (*proto.MessageResponse, error) {
	msgID, err := strconv.ParseUint(req.Id, 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid message id: %v", err)
	}

	message, err := s.messageRepo.UpdateMessage(ctx, uint(msgID), req.Role, req.Content)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update message: %v", err)
	}

	return &proto.MessageResponse{
		Id:        fmt.Sprintf("%d", message.ID),
		ChatId:    message.ChatID,
		Role:      message.Role,
		Content:   message.Content,
		Platform:  req.Platform,
		CreatedAt: message.Time.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) DeleteMessage(ctx context.Context, req *proto.DeleteMessageRequest) (*proto.DeleteMessageResponse, error) {
	msgID, err := strconv.ParseUint(req.Id, 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid message id: %v", err)
	}

	err = s.messageRepo.DeleteMessage(ctx, uint(msgID))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete message: %v", err)
	}

	return &proto.DeleteMessageResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) GetChatPagesCount(ctx context.Context, req *proto.GetChatPagesCountRequest) (*proto.ChatPagesCountResponse, error) {
	pagesCount, err := s.chatRepo.GetChatPagesCount(ctx, req.AssistantId, req.ChatsPerPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chat pages count: %v", err)
	}

	return &proto.ChatPagesCountResponse{
		PagesCount: pagesCount,
	}, nil
}

func (s *DatabaseServer) GetChatPage(ctx context.Context, req *proto.GetChatPageRequest) (*proto.ChatsResponse, error) {
	chats, err := s.chatRepo.GetChatPage(ctx, req.AssistantId, req.Page, req.ChatsPerPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chat page: %v", err)
	}

	var protoChats []*proto.ChatResponse
	for _, chat := range chats {
		protoChats = append(protoChats, &proto.ChatResponse{
			Id:          chat.ID,
			AssistantId: chat.AssistantID,
			CustomerId: func() string {
				if chat.CustomerID != nil {
					return *chat.CustomerID
				}
				return ""
			}(),
			CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
			MessageCount: chat.MessageCount,
		})
	}

	return &proto.ChatsResponse{
		Chats: protoChats,
	}, nil
}

func (s *DatabaseServer) GetChatPagesCountByUserID(ctx context.Context, req *proto.GetChatPagesCountByUserIDRequest) (*proto.ChatPagesCountResponse, error) {
	pagesCount, err := s.chatRepo.GetChatPagesCountByUserID(ctx, req.AssistantIds, req.ChatsPerPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chat pages count by user: %v", err)
	}

	return &proto.ChatPagesCountResponse{
		PagesCount: pagesCount,
	}, nil
}

func (s *DatabaseServer) GetChatPageByUserID(ctx context.Context, req *proto.GetChatPageByUserIDRequest) (*proto.ChatsResponse, error) {
	log.Printf("[gRPC] GetChatPageByUserID: AssistantIds=%v, Page=%d, ChatsPerPage=%d", req.AssistantIds, req.Page, req.ChatsPerPage)
	chats, err := s.chatRepo.GetChatPageByUserID(ctx, req.AssistantIds, req.Page, req.ChatsPerPage)
	if err != nil {
		log.Printf("[gRPC] GetChatPageByUserID Error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get chat page by user: %v", err)
	}

	log.Printf("[gRPC] GetChatPageByUserID: Returning %d chats", len(chats))

	var protoChats []*proto.ChatResponse
	for _, chat := range chats {
		customerId := ""
		if chat.CustomerID != nil {
			customerId = *chat.CustomerID
		}

		protoChats = append(protoChats, &proto.ChatResponse{
			Id:           chat.ID,
			AssistantId:  chat.AssistantID,
			CustomerId:   customerId,
			Platform:     "", // Platform not stored in chat model
			CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
			MessageCount: chat.MessageCount,
		})
	}

	return &proto.ChatsResponse{
		Chats: protoChats,
	}, nil
}

func (s *DatabaseServer) SearchChatsByCustomer(ctx context.Context, req *proto.SearchChatsByCustomerRequest) (*proto.SearchChatsByCustomerResponse, error) {
	chats, totalCount, err := s.chatRepo.SearchChatsByCustomer(ctx, req.AssistantIds, req.Search)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search chats: %v", err)
	}

	var protoChats []*proto.ChatResponse
	for _, chat := range chats {
		customerId := ""
		if chat.CustomerID != nil {
			customerId = *chat.CustomerID
		}

		protoChats = append(protoChats, &proto.ChatResponse{
			Id:           chat.ID,
			AssistantId:  chat.AssistantID,
			CustomerId:   customerId,
			Platform:     "", // Platform not stored in chat model
			CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
			MessageCount: chat.MessageCount,
		})
	}

	return &proto.SearchChatsByCustomerResponse{
		Chats:      protoChats,
		TotalCount: totalCount,
	}, nil
}

func (s *DatabaseServer) GetAssistant(ctx context.Context, req *proto.GetAssistantRequest) (*proto.AssistantResponse, error) {
	assistant, err := s.assistantRepo.GetAssistantByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "assistant not found: %v", err)
	}

	return &proto.AssistantResponse{
		Id:               assistant.ID,
		Name:             assistant.Name,
		ApiToken:         assistant.APIToken,
		UserId:           assistant.UserID,
		CreatedAt:        assistant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        assistant.UpdatedAt.Format(time.RFC3339),
		Configuration:    assistant.Configuration,
		TelegramBotToken: assistant.TelegramBotToken,
		Type:             assistant.Type,
	}, nil
}

func (s *DatabaseServer) GetAssistantByAPIToken(ctx context.Context, req *proto.GetAssistantByAPITokenRequest) (*proto.AssistantResponse, error) {
	assistant, err := s.assistantRepo.GetAssistantByAPIToken(ctx, req.ApiToken)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "assistant not found: %v", err)
	}

	return &proto.AssistantResponse{
		Id:               assistant.ID,
		Name:             assistant.Name,
		ApiToken:         assistant.APIToken,
		UserId:           assistant.UserID,
		CreatedAt:        assistant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        assistant.UpdatedAt.Format(time.RFC3339),
		Configuration:    assistant.Configuration,
		TelegramBotToken: assistant.TelegramBotToken,
		Type:             assistant.Type,
	}, nil
}

func (s *DatabaseServer) UpdateAssistant(ctx context.Context, req *proto.UpdateAssistantRequest) (*proto.AssistantResponse, error) {
	assistant, err := s.assistantRepo.UpdateAssistant(ctx, req.Id, req.Name, req.Configuration, req.ApiToken, req.TelegramBotToken, req.Type)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update assistant: %v", err)
	}

	return &proto.AssistantResponse{
		Id:               assistant.ID,
		Name:             assistant.Name,
		ApiToken:         assistant.APIToken,
		UserId:           assistant.UserID,
		CreatedAt:        assistant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        assistant.UpdatedAt.Format(time.RFC3339),
		Configuration:    assistant.Configuration,
		TelegramBotToken: assistant.TelegramBotToken,
		Type:             assistant.Type,
	}, nil
}

func (s *DatabaseServer) DeleteAssistant(ctx context.Context, req *proto.DeleteAssistantRequest) (*proto.DeleteAssistantResponse, error) {
	err := s.assistantRepo.DeleteAssistant(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete assistant: %v", err)
	}

	return &proto.DeleteAssistantResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) GetAssistantsByUserID(ctx context.Context, req *proto.GetAssistantsByUserIDRequest) (*proto.AssistantsResponse, error) {
	assistants, err := s.assistantRepo.GetAssistantsByUserID(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get assistants by user id: %v", err)
	}

	var protoAssistants []*proto.AssistantResponse
	for _, assistant := range assistants {
		protoAssistants = append(protoAssistants, &proto.AssistantResponse{
			Id:               assistant.ID,
			Name:             assistant.Name,
			ApiToken:         assistant.APIToken,
			UserId:           assistant.UserID,
			CreatedAt:        assistant.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        assistant.UpdatedAt.Format(time.RFC3339),
			Configuration:    assistant.Configuration,
			TelegramBotToken: assistant.TelegramBotToken,
			Type:             assistant.Type,
		})
	}

	return &proto.AssistantsResponse{
		Assistants: protoAssistants,
	}, nil
}
func (s *DatabaseServer) GetLatestChatByCustomer(ctx context.Context, req *proto.GetLatestChatByCustomerRequest) (*proto.ChatResponse, error) {
	chat, err := s.chatRepo.GetLatestChatByCustomer(ctx, req.AssistantId, req.CustomerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get latest chat: %v", err)
	}

	if chat == nil {
		return &proto.ChatResponse{}, nil
	}

	return &proto.ChatResponse{
		Id:          chat.ID,
		AssistantId: chat.AssistantID,
		CustomerId: func() string {
			if chat.CustomerID != nil {
				return *chat.CustomerID
			}
			return ""
		}(),
		CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
		MessageCount: chat.MessageCount,
	}, nil
}

func (s *DatabaseServer) SaveTwilioConfig(ctx context.Context, req *proto.SaveTwilioConfigRequest) (*proto.TwilioConfigResponse, error) {
	config := &models.TwilioConfig{
		AssistantID:  req.AssistantId,
		UserID:       req.UserId,
		TwilioNumber: req.TwilioNumber,
		AccountSID:   req.AccountSid,
		AuthToken:    req.AuthToken,
	}

	if err := s.twilioRepo.SaveConfig(config); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save twilio config: %v", err)
	}

	return &proto.TwilioConfigResponse{
		AssistantId:  config.AssistantID,
		UserId:       config.UserID,
		TwilioNumber: config.TwilioNumber,
		AccountSid:   config.AccountSID,
		AuthToken:    config.AuthToken,
		CreatedAt:    config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    config.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) GetTwilioConfig(ctx context.Context, req *proto.GetTwilioConfigRequest) (*proto.TwilioConfigResponse, error) {
	config, err := s.twilioRepo.GetConfigByAssistantID(req.AssistantId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "twilio config not found: %v", err)
	}

	return &proto.TwilioConfigResponse{
		AssistantId:  config.AssistantID,
		UserId:       config.UserID,
		TwilioNumber: config.TwilioNumber,
		AccountSid:   config.AccountSID,
		AuthToken:    config.AuthToken,
		CreatedAt:    config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    config.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DatabaseServer) DeleteTwilioConfig(ctx context.Context, req *proto.DeleteTwilioConfigRequest) (*proto.DeleteTwilioConfigResponse, error) {
	if err := s.twilioRepo.DeleteConfig(req.AssistantId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete twilio config: %v", err)
	}

	return &proto.DeleteTwilioConfigResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) GetCampusloginByUserId(ctx context.Context, req *proto.CampusloginRequest) (*proto.CampusloginResponse, error) {
	record, err := s.campusloginRepo.GetByUserId(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "campuslogin not found: %v", err)
	}

	return &proto.CampusloginResponse{
		UserId:                 record.UserId,
		ContactId:              int32(record.ContactID),
		ProgramId:              int32(record.ProgramID),
		IsGrade11OrLower:       record.IsGrade11OrLower,
		IsInternationalStudent: record.IsInternationalStudent,
	}, nil
}

func (s *DatabaseServer) UpsertCampuslogin(ctx context.Context, req *proto.UpsertCampusloginRequest) (*proto.UpsertCampusloginResponse, error) {
	campuslogin := &models.Campuslogin{
		UserId:    req.UserId,
		ContactID: int(req.ContactId),
		ProgramID: int(req.ProgramId),
	}

	if err := s.campusloginRepo.Upsert(campuslogin); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert campuslogin: %v", err)
	}

	return &proto.UpsertCampusloginResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) DeleteAllChatsAndMessages(ctx context.Context, req *proto.DeleteAllChatsAndMessagesRequest) (*proto.DeleteAllChatsAndMessagesResponse, error) {
	if err := s.messageRepo.DeleteAllMessages(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete all messages: %w", err)
	}
	if err := s.chatRepo.DeleteAllChats(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete all chats: %w", err)
	}

	return &proto.DeleteAllChatsAndMessagesResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) DeleteChatAndMessages(ctx context.Context, req *proto.DeleteChatAndMessagesRequest) (*proto.DeleteChatAndMessagesResponse, error) {
	if err := s.messageRepo.DeleteMessagesByChatID(ctx, req.ChatId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete messages for chat %s: %v", req.ChatId, err)
	}
	if err := s.chatRepo.DeleteChat(ctx, req.ChatId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete chat %s: %v", req.ChatId, err)
	}

	return &proto.DeleteChatAndMessagesResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) UpdateChatIsEnd(ctx context.Context, req *proto.UpdateChatIsEndRequest) (*proto.ChatResponse, error) {
	chat, err := s.chatRepo.UpdateChatIsEnd(ctx, req.Id, req.IsEnd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update chat is_end: %v", err)
	}

	return &proto.ChatResponse{
		Id:          chat.ID,
		AssistantId: chat.AssistantID,
		CustomerId: func() string {
			if chat.CustomerID != nil {
				return *chat.CustomerID
			}
			return ""
		}(),
		CreatedAt:    chat.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    chat.UpdatedAt.Format(time.RFC3339),
		MessageCount: chat.MessageCount,
		IsEnd:        chat.IsEnd,
	}, nil
}

func (s *DatabaseServer) SetCampusloginFlags(ctx context.Context, req *proto.SetCampusloginFlagsRequest) (*proto.SetCampusloginFlagsResponse, error) {
	if err := s.campusloginRepo.SetFlags(req.UserId, req.IsGrade11OrLower, req.IsInternationalStudent); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set campuslogin flags: %v", err)
	}

	return &proto.SetCampusloginFlagsResponse{
		Success: true,
	}, nil
}

func (s *DatabaseServer) UpdateChatFollowupStage(ctx context.Context, req *proto.UpdateChatFollowupStageRequest) (*proto.ChatResponse, error) {
	chat, err := s.chatRepo.UpdateChatFollowupStage(ctx, req.Id, int(req.Stage))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update chat followup stage: %v", err)
	}

	return &proto.ChatResponse{
		Id:          chat.ID,
		AssistantId: chat.AssistantID,
		CustomerId: func() string {
			if chat.CustomerID != nil {
				return *chat.CustomerID
			}
			return ""
		}(),
		CreatedAt:     chat.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     chat.UpdatedAt.Format(time.RFC3339),
		MessageCount:  chat.MessageCount,
		IsEnd:         chat.IsEnd,
		FollowupStage: int32(chat.FollowupStage),
	}, nil
}

func (s *DatabaseServer) GetChatsForFollowup(ctx context.Context, req *proto.GetChatsForFollowupRequest) (*proto.ChatsResponse, error) {
	chats, err := s.chatRepo.GetChatsForFollowup(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chats for followup: %v", err)
	}

	var protoChats []*proto.ChatResponse
	for _, chat := range chats {
		customerId := ""
		if chat.CustomerID != nil {
			customerId = *chat.CustomerID
		}

		protoChats = append(protoChats, &proto.ChatResponse{
			Id:            chat.ID,
			AssistantId:   chat.AssistantID,
			CustomerId:    customerId,
			CreatedAt:     chat.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     chat.UpdatedAt.Format(time.RFC3339),
			MessageCount:  chat.MessageCount,
			IsEnd:         chat.IsEnd,
			FollowupStage: int32(chat.FollowupStage),
		})
	}

	return &proto.ChatsResponse{
		Chats: protoChats,
	}, nil
}
