package main

import (
	context "context"
	"log/slog"
	"net"

	"github.com/Boar-D-White-Foundation/kvgo"
	pb "github.com/Boar-D-White-Foundation/kvgo/proto"
	"google.golang.org/grpc"
)

type rafterServer struct {
	pb.UnimplementedRafterServer
	clients []pb.RafterClient
	cfg     *kvgo.Config
	raft    *kvgo.Raft
}

// HandleAppendEntryRequest implements kvgo.RequestHandler.
func (r *rafterServer) HandleAppendEntryRequest(request kvgo.AppendEntryRequest) {
	go func() {
		req := pb.AppendEntryRequest{
			Id:   int32(request.Id),
			Term: request.Term,
		}

		conn, err := grpc.Dial(r.cfg.Cluster[request.ReceiverId])
		if err != nil {
			slog.Error("failed to create client: %s", r.cfg.Cluster[request.ReceiverId])
			return
		}

		rc := pb.NewRafterClient(conn)

		//recipient := r.clients[request.ReceiverId]
		resp, err := rc.AppendEntry(context.Background(), &req)

		if err != nil {
			slog.Error("can't append entry: %s", err)
			return
		}

		r.raft.SendAppendEntryResponse(kvgo.AppendEntryResponse{
			Term:          resp.Term,
			Success:       resp.Success,
			RequestInTerm: request.Term,
			ReceiverId:    request.ReceiverId,
			MatchIndex:    request.PrevLogIndex + len(request.Entries),
		})
	}()
}

// HandleVoteRequest implements kvgo.RequestHandler.
func (r *rafterServer) HandleVoteRequest(request kvgo.VoteRequest) {
	go func() {
		req := pb.VoteRequest{
			CandidateId:   int32(request.CandidateId),
			RequestInTerm: request.Term,
		}

		conn, err := grpc.Dial(r.cfg.Cluster[request.ReceiverId])
		if err != nil {
			slog.Error("failed to create client: %s", r.cfg.Cluster[request.ReceiverId])
			return
		}

		rc := pb.NewRafterClient(conn)

		//recipient := r.clients[request.ReceiverId]
		resp, err := rc.Vote(context.Background(), &req)

		if err != nil {
			slog.Error("can't vote: %s", err)
			return
		}

		r.raft.SendVoteResponse(kvgo.VoteResponse{
			ReceiverId:    request.ReceiverId,
			RequestInTerm: request.Term,
			Term:          resp.Term,
			VoteGranted:   resp.VoteGranted,
		})
	}()
}

func makeGrpcServer(cfg *kvgo.Config, raft *kvgo.Raft) *rafterServer {
	return &rafterServer{
		cfg:  cfg,
		raft: raft,
	}
}

// fmt.Sprintf(":%d", *port)
func (r *rafterServer) Start() {
	lis, err := net.Listen("tcp", r.cfg.Cluster[r.cfg.CurrentId])
	if err != nil {
		slog.Error("failed to listen:", err)
	}

	s := grpc.NewServer()
	//create clients for all ports in the cluster
	/*clients := make([]pb.RafterClient, len(r.cfg.Cluster))
	for i := 0; i < len(r.cfg.Cluster); i++ {
		port := r.cfg.Cluster[i]

		conn, err := grpc.Dial(port)
		if err != nil {
			slog.Error("failed to create client: %s", port)
		}

		rc := pb.NewRafterClient(conn)
		clients[i] = rc

	}

	r.clients = clients*/

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
