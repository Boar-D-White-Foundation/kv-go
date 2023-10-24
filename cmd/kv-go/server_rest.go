package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Boar-D-White-Foundation/kvgo"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
)

type appendEntryRequestDto struct {
	Id   int    `json:"id"`
	Term uint64 `json:"term"`
}

type appendEntryResponseDto struct {
	Term    uint64 `json:"term"`
	Success bool   `json:"success"`
}

type voteRequestDto struct {
	Term        uint64 `json:"term"`
	CandidateId int    `json:"candidateId"`
}

type voteResponseDto struct {
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"voteGranted"`
}

type restServer struct {
	cfg  *kvgo.Config
	raft *kvgo.Raft
}

func makeRestServer(cfg *kvgo.Config, raft *kvgo.Raft) restServer {
	return restServer{
		cfg:  cfg,
		raft: raft,
	}
}

func (r *restServer) StartHttpServer() {
	router := gin.Default()
	router.POST("/appendEntryRequest", r.postAppendEntryRequest)
	router.POST("/voteRequest", r.postVoteRequest)

	err := router.Run(r.cfg.Cluster[r.cfg.CurrentId])
	if err != nil {
		slog.Error("router finished with error", "error", err)
	}
}

func (r *restServer) postAppendEntryRequest(c *gin.Context) {
	var request appendEntryRequestDto
	if err := c.BindJSON(&request); err != nil {
		slog.Error("can't read request body", "error", err)
	}

	message := kvgo.AppendEntryMessage{
		Request: kvgo.AppendEntryRequest{
			Id:         request.Id,
			ReceiverId: r.cfg.CurrentId,
			Term:       request.Term,
		},
		Response: make(chan kvgo.AppendEntryResponse),
	}

	r.raft.SendAppendEntryMessage(message)
	response := <-message.Response
	c.IndentedJSON(http.StatusOK, &appendEntryResponseDto{
		Term:    response.Term,
		Success: response.Success,
	})
}

func (r *restServer) postVoteRequest(c *gin.Context) {
	var request voteRequestDto
	if err := c.BindJSON(&request); err != nil {
		slog.Error("can't read request body", "error", err)
	}

	message := kvgo.VoteMessage{
		Request: kvgo.VoteRequest{
			ReceiverId:  r.cfg.CurrentId,
			Term:        request.Term,
			CandidateId: request.CandidateId,
		},
		Response: make(chan kvgo.VoteResponse),
	}

	r.raft.SendVoteMessage(message)
	response := <-message.Response
	c.IndentedJSON(http.StatusOK, &voteResponseDto{
		Term:        response.Term,
		VoteGranted: response.VoteGranted,
	})
}

func (r *restServer) HandleAppendEntryRequest(request kvgo.AppendEntryRequest) {
	go func() {
		url := fmt.Sprintf("http://%s/appendEntryRequest", r.cfg.Cluster[request.ReceiverId])
		requestBody, err := json.Marshal(&appendEntryRequestDto{
			Id:   request.Id,
			Term: request.Term,
		})
		if err != nil {
			slog.Error("can't marshal AppendEntryRequest", "error", err)
			return
		}
		response, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
		if err != nil {
			slog.Error("can't send AppendEntryRequest", "error", err)
			return
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Error("can't read AppendEntryResponse body", "error", err)
			return
		}

		var resp appendEntryResponseDto
		err = json.Unmarshal(body, &resp)
		if err != nil {
			slog.Error("can't unmarshal AppendEntryResponse body", "error", err)
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

func (r *restServer) HandleVoteRequest(request kvgo.VoteRequest) {
	go func() {
		url := fmt.Sprintf("http://%s/voteRequest", r.cfg.Cluster[request.ReceiverId])
		requestBody, err := json.Marshal(&voteRequestDto{
			Term:        request.Term,
			CandidateId: request.CandidateId,
		})
		if err != nil {
			slog.Error("can't marshal VoteRequest", "error", err)
			return
		}
		response, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
		if err != nil {
			slog.Error("can't send VoteRequest", "error", err)
			return
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Error("can't read VoteResponse body", "error", err)
			return
		}

		var resp voteResponseDto
		err = json.Unmarshal(body, &resp)
		if err != nil {
			slog.Error("can't unmarshal VoteResponse body", "error", err)
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
