package grpc

import (
	"context"

	"github.com/rs/zerolog"

	pb "github.com/Conte777/NewsFlow/pkg/proto/subscription/v1"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
)

// Server implements the gRPC SubscriptionService
type Server struct {
	pb.UnimplementedSubscriptionServiceServer
	useCase domain.SubscriptionUseCase
	logger  zerolog.Logger
}

// NewServer creates a new gRPC server instance
func NewServer(useCase domain.SubscriptionUseCase, logger zerolog.Logger) *Server {
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
