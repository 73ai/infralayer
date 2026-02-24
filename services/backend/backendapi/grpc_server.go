package backendapi

import (
	"context"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/backendapi/proto"
	"google.golang.org/grpc"
)

type grpcServer struct {
	proto.UnimplementedBackendServiceServer
	svc backend.ConversationService
}

func NewGRPCServer(svc backend.ConversationService) *grpc.Server {
	server := grpc.NewServer()
	proto.RegisterBackendServiceServer(server, &grpcServer{
		svc: svc,
	})
	return server
}

func (s *grpcServer) SendReply(ctx context.Context, req *proto.SendReplyCommand) (*proto.Status, error) {
	err := s.svc.SendReply(ctx, backend.SendReplyCommand{
		ConversationID: req.ConversationId,
		Message:        req.Message,
	})

	if err != nil {
		return &proto.Status{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &proto.Status{
		Success: true,
		Error:   "",
	}, nil
}
