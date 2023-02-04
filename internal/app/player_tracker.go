package app

import (
	"context"
	"fmt"
	"github.com/emortalmc/proto-specs/gen/go/grpc/playertracker"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"net"
	"player-tracker/internal/config"
	"player-tracker/internal/rabbitmq"
	"player-tracker/internal/rabbitmq/listener"
	"player-tracker/internal/repository"
	"player-tracker/internal/service"
)

func Run(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger) {
	repo, err := repository.NewMongoRepository(ctx, cfg.MongoDB)
	if err != nil {
		logger.Fatalw("failed to create repository", err)
	}

	// NOTE: We can share a RabbitMQ connection, but it is not recommended to share a channel
	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ)
	if err != nil {
		logger.Fatalw("failed to create rabbitmq connection", "error", err)
	}

	err = listener.NewRabbitMQListener(logger, repo, rabbitConn)
	if err != nil {
		logger.Fatalw("failed to create rabbitmq listener", "error", err)
	}
	logger.Infow("connected to RabbitMQ", "host", cfg.RabbitMQ.Host)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		logger.Fatalw("failed to listen", "error", err)
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_zap.UnaryServerInterceptor(logger.Desugar(), grpc_zap.WithLevels(func(code codes.Code) zapcore.Level {
				if code == codes.OK {
					return zapcore.DebugLevel
				}
				return zapcore.InfoLevel
			})),
			grpc_prometheus.UnaryServerInterceptor,
		),
	)
	playertracker.RegisterPlayerTrackerServer(s, service.NewPlayerTrackerService(repo))
	logger.Infow("listening on port", "port", cfg.Port)

	err = s.Serve(lis)
	if err != nil {
		logger.Fatalw("failed to serve", "error", err)
	}
}
