package service

import (
	"context"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/playertracker"
	"github.com/emortalmc/proto-specs/gen/go/model/common"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/player_tracker"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"player-tracker/internal/repository"
	"strings"
)

var (
	serverTypeToFleet = map[common.ServerType]string{
		common.ServerType_LOBBY:           "lobby",
		common.ServerType_MARATHON:        "marathon",
		common.ServerType_BLOCK_SUMO:      "block-sumo",
		common.ServerType_PARKOURTAG:      "parkourtag",
		common.ServerType_LAZERTAG:        "lazertag",
		common.ServerType_HOLEY_MOLEY:     "holey-moley",
		common.ServerType_MARATHON_RACING: "marathon-racing",
		common.ServerType_BATTLE:          "battle",
		common.ServerType_MINESWEEPER:     "minesweeper",
	}
)

type playerTrackerService struct {
	pb.PlayerTrackerServer

	repo repository.Repository
}

func NewPlayerTrackerService(repo repository.Repository) pb.PlayerTrackerServer {
	return &playerTrackerService{
		repo: repo,
	}
}

func (s *playerTrackerService) GetPlayerServer(ctx context.Context, req *pb.GetPlayerServerRequest) (*pb.GetPlayerServerResponse, error) {
	pId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player id")
	}

	p, err := s.repo.GetPlayer(ctx, pId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.GetPlayerServerResponse{Server: nil}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get player from repository: %v", err)
	}

	return &pb.GetPlayerServerResponse{Server: &pbmodel.PlayerLocation{
		ServerId: p.GameServerId,
		ProxyId:  p.ProxyId,
	}}, nil
}

func (s *playerTrackerService) GetPlayerServers(ctx context.Context, req *pb.GetPlayerServersRequest) (*pb.GetPlayerServersResponse, error) {
	pIds := make([]uuid.UUID, len(req.PlayerIds))
	for i, pId := range req.PlayerIds {
		parsed, err := uuid.Parse(pId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid player id")
		}
		pIds[i] = parsed
	}

	players, err := s.repo.GetPlayers(ctx, pIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get players from repository: %v", err)
	}

	locations := make(map[string]*pbmodel.PlayerLocation, len(players))
	for _, p := range players {
		locations[p.Id.String()] = &pbmodel.PlayerLocation{
			ServerId: p.GameServerId,
			ProxyId:  p.ProxyId,
		}
	}

	return &pb.GetPlayerServersResponse{PlayerServers: locations}, nil
}

func (s *playerTrackerService) GetServerPlayers(ctx context.Context, req *pb.GetServerPlayersRequest) (*pb.GetServerPlayersResponse, error) {
	players, err := s.repo.GetServerPlayers(ctx, req.ServerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get server players from repository: %v", err)
	}

	protoPlayers := make([]*pbmodel.OnlinePlayer, len(players))
	for i, p := range players {
		protoPlayers[i] = &pbmodel.OnlinePlayer{
			PlayerId: p.Id.String(),
			Username: p.Username,
		}
	}

	return &pb.GetServerPlayersResponse{OnlinePlayers: protoPlayers}, nil
}

func (s *playerTrackerService) GetServerPlayerCount(ctx context.Context, req *pb.GetServerPlayerCountRequest) (*pb.GetServerPlayerCountResponse, error) {
	proxy := strings.HasPrefix(req.ServerId, "proxy-")

	count, err := s.repo.GetServerPlayerCount(ctx, req.ServerId, proxy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get server player count from repository: %v", err)
	}

	return &pb.GetServerPlayerCountResponse{PlayerCount: uint32(count)}, nil
}

func (s *playerTrackerService) GetServerTypePlayerCount(ctx context.Context, req *pb.GetServerTypePlayerCountRequest) (*pb.ServerTypePlayerCountResponse, error) {
	if req.ServerType == common.ServerType_PROXY {
		count, err := s.repo.PlayerCount(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get player count from repository: %v", err)
		}
		return &pb.ServerTypePlayerCountResponse{PlayerCount: uint32(count)}, nil
	}

	fleet, ok := serverTypeToFleet[req.ServerType]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unknown server type %v", req.ServerType)
	}
	count, err := s.repo.GetServerTypePlayerCount(ctx, fleet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get server type player count from repository: %v", err)
	}

	return &pb.ServerTypePlayerCountResponse{PlayerCount: uint32(count)}, nil
}

func (s *playerTrackerService) GetServerTypesPlayerCount(ctx context.Context, req *pb.GetServerTypesPlayerCountRequest) (*pb.ServerTypesPlayerCountResponse, error) {
	counts := make(map[int32]uint32, len(req.ServerTypes))
	for _, t := range req.ServerTypes {
		if t == common.ServerType_PROXY {
			count, err := s.repo.PlayerCount(ctx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get player count from repository: %v", err)
			}
			counts[int32(common.ServerType_PROXY.Number())] = uint32(count)
			continue
		}
		fleet, ok := serverTypeToFleet[t]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "unknown server type %v", t)
		}

		count, err := s.repo.GetServerTypePlayerCount(ctx, fleet)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get server type player count from repository: %v", err)
		}

		cardinal := int32(t.Number())
		log.Printf("cardinal: %v for %v", cardinal, t)

		counts[cardinal] = uint32(count)
	}

	return &pb.ServerTypesPlayerCountResponse{PlayerCounts: counts}, nil
}
