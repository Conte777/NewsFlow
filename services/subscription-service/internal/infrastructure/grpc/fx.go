package grpc

import (
	"context"
	"net"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"google.golang.org/grpc"

	pb "github.com/Conte777/NewsFlow/pkg/proto/subscription/v1"
	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	grpcDelivery "github.com/Conte777/NewsFlow/services/subscription-service/internal/delivery/grpc"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/deps"
)

var Module = fx.Module(
	"grpc",
	fx.Provide(NewGRPCServer),
	fx.Invoke(registerGRPCServer),
)

type GRPCServerResult struct {
	fx.Out
	Server  *grpc.Server
	Handler *grpcDelivery.Server
}

func NewGRPCServer(useCase deps.SubscriptionUseCase, logger zerolog.Logger) GRPCServerResult {
	server := grpc.NewServer()
	handler := grpcDelivery.NewServer(useCase, logger)
	pb.RegisterSubscriptionServiceServer(server, handler)

	return GRPCServerResult{
		Server:  server,
		Handler: handler,
	}
}

func registerGRPCServer(
	lc fx.Lifecycle,
	cfg *config.ServiceConfig,
	server *grpc.Server,
	log zerolog.Logger,
) error {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
			if err != nil {
				log.Error().Err(err).Str("port", cfg.GRPCPort).Msg("failed to listen for gRPC")
				return err
			}

			go func() {
				log.Info().Str("port", cfg.GRPCPort).Msg("gRPC server started")
				if err := server.Serve(lis); err != nil {
					log.Error().Err(err).Msg("gRPC server failed")
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("stopping gRPC server...")
			server.GracefulStop()
			log.Info().Msg("gRPC server stopped")
			return nil
		},
	})

	return nil
}
