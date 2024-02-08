package main

import (
	context "context"
	"fmt"
	"log/slog"
	"net"

	"github.com/Boar-D-White-Foundation/kvgo"
	pb "github.com/Boar-D-White-Foundation/kvgo/proto"
	"google.golang.org/grpc"
)

type rafterServer struct {
	pb.UnimplementedRafterServer
	cfg  *kvgo.Config
	raft *kvgo.Raft
}

func makeGrpcServer(cfg *kvgo.Config, raft *kvgo.Raft) rafterServer {
	return rafterServer{
		cfg:  cfg,
		raft: raft,
	}
}

func Start(r *rafterServer) {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", r.cfg.Cluster[r.cfg.CurrentId]))
	if err != nil {
		slog.Error("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterRafterServer(s, &rafterServer{})
	slog.Info("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve: %v", err)
	}
}

func (s *rafterServer) AppendEntry(ctx context.Context, request *pb.AppendEntryRequest) (*pb.AppendEntryResponse, error) {
	message := kvgo.AppendEntryMessage{
		Request: kvgo.AppendEntryRequest{
			Id:         int(request.Id),
			ReceiverId: s.cfg.CurrentId,
			Term:       request.Term,
		},
		Response: make(chan kvgo.AppendEntryResponse),
	}

	s.raft.SendAppendEntryMessage(message)
	response := <-message.Response
	return &pb.AppendEntryResponse{
		ReceiverId:    int32(response.ReceiverId),
		RequestInTerm: response.RequestInTerm,
		MatchIndex:    int32(response.MatchIndex),
		Term:          response.Term,
		Success:       response.Success,
	}, nil
}

func (s *rafterServer) Vote(ctx context.Context, request *pb.VoteRequest) (*pb.VoteResponse, error) {
	message := kvgo.VoteMessage{
		Request: kvgo.VoteRequest{
			ReceiverId:  s.cfg.CurrentId,
			Term:        request.RequestInTerm,
			CandidateId: int(request.CandidateId),
		},
		Response: make(chan kvgo.VoteResponse),
	}
	s.raft.SendVoteMessage(message)

	response := <-message.Response

	return &pb.VoteResponse{
		Term:        response.Term,
		VoteGranted: response.VoteGranted,
	}, nil
}
