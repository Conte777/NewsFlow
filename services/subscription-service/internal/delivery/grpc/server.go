package grpc

import (
	"context"

	"github.com/rs/zerolog"

	pb "github.com/Conte777/NewsFlow/pkg/proto/subscription/v1"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/deps"
)

// Server implements the gRPC SubscriptionService
type Server struct {
	pb.UnimplementedSubscriptionServiceServer
	useCase deps.SubscriptionUseCase
	logger  zerolog.Logger
}

// NewServer creates a new gRPC server instance
func NewServer(useCase deps.SubscriptionUseCase, logger zerolog.Logger) *Server {
	return &Server{
		useCase: useCase,
		logger:  logger,
	}
}

// GetUserSubscriptions returns all active subscriptions for a user
func (s *Server) GetUserSubscriptions(ctx context.Context, req *pb.GetUserSubscriptionsRequest) (*pb.GetUserSubscriptionsResponse, error) {
	s.logger.Debug().
		Int64("user_id", req.UserId).
		Msg("gRPC GetUserSubscriptions called")

	subs, err := s.useCase.GetUserSubscriptions(ctx, req.UserId)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int64("user_id", req.UserId).
			Msg("Failed to get user subscriptions")
		return nil, err
	}

	result := make([]*pb.Subscription, len(subs))
	for i, sub := range subs {
		result[i] = &pb.Subscription{
			UserId:      sub.UserID,
			ChannelId:   sub.ChannelID,
			ChannelName: sub.ChannelName,
			IsActive:    sub.IsActive,
		}
	}

	s.logger.Debug().
		Int64("user_id", req.UserId).
		Int("count", len(result)).
		Msg("Returning user subscriptions")

	return &pb.GetUserSubscriptionsResponse{Subscriptions: result}, nil
}

// GetChannelSubscribers returns all users subscribed to a channel
func (s *Server) GetChannelSubscribers(ctx context.Context, req *pb.GetChannelSubscribersRequest) (*pb.GetChannelSubscribersResponse, error) {
	s.logger.Debug().
		Str("channel_id", req.ChannelId).
		Msg("gRPC GetChannelSubscribers called")

	userIDs, err := s.useCase.GetChannelSubscribers(ctx, req.ChannelId)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("channel_id", req.ChannelId).
			Msg("Failed to get channel subscribers")
		return nil, err
	}

	s.logger.Debug().
		Str("channel_id", req.ChannelId).
		Int("count", len(userIDs)).
		Msg("Returning channel subscribers")

	return &pb.GetChannelSubscribersResponse{UserIds: userIDs}, nil
}
