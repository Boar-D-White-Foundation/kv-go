package main

import (
	context "context"
	"log/slog"
	"net/http"
	pb "raft"

	"github.com/Boar-D-White-Foundation/kvgo"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type rafterServer struct {
	pb.UnimplementedRafterServer
}

func (s *rafterServer) AppendEntry(ctx context.Context, request *AppendEntryRequest) (*AppendEntryResponse, error) {
	
	if err := c.BindJSON(&request); err != nil {
		slog.Error("can't read request body", "error", err)
	}

	message := kvgo.AppendEntryMessage{
		Request: kvgo.AppendEntryRequest{
			Id:         request.id,
			ReceiverId: s.cfg.CurrentId,
			Term:       request.,
		},
		Response: make(chan kvgo.AppendEntryResponse),
	}

	r.raft.SendAppendEntryMessage(message)
	response := <-message.Response
	c.IndentedJSON(http.StatusOK, &appendEntryResponseDto{
		Term:    response.Term,
		Success: response.Success,
	})
	return &pb.
	return nil, status.Errorf(codes.Unimplemented, "method AppendEntry not implemented")
}
func (s *rafterServer) Vote(context.Context, *VoteRequest) (*VoteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Vote not implemented")
}
