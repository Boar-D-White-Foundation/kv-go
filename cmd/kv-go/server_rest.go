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

func (r *Raft) StartHttpServer() {
	router := gin.Default()
	router.POST("/appendEntryRequest", r.postAppendEntryRequest)
	router.POST("/voteRequest", r.postVoteRequest)

	err := router.Run(r.cfg.Cluster[r.cfg.CurrentId])
	if err != nil {
		slog.Error("router finished with error", "error", err)
	}
}

func (r *Raft) postAppendEntryRequest(c *gin.Context) {
	var request kvgo.AppendEntryRequest
	if err := c.BindJSON(&request); err != nil {
		slog.Error("can't read request body", "error", err)
	}

	message := appendEntryMessage{
		request:  request,
		response: make(chan kvgo.AppendEntryResponse),
	}

	r.appendEntries <- message
	response := <-message.response
	c.IndentedJSON(http.StatusOK, &response)
}

func (r *Raft) postVoteRequest(c *gin.Context) {
	var request kvgo.VoteRequest
	if err := c.BindJSON(&request); err != nil {
		slog.Error("can't read request body", "error", err)
	}

	message := voteMessage{
		request:  request,
		response: make(chan kvgo.VoteResponse),
	}

	r.votes <- message
	response := <-message.response
	c.IndentedJSON(http.StatusOK, &response)
}

func (r *Raft) HandleAppendEntryRequest(receiverId int, request kvgo.AppendEntryRequest) {
	go func() {
		url := fmt.Sprintf("http://%s/appendEntryRequest", r.cfg.Cluster[receiverId])
		requestBody, err := json.Marshal(&request)
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

		var resp kvgo.AppendEntryResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			slog.Error("can't unmarshal AppendEntryResponse body", "error", err)
			return
		}

		r.appendEntryResponses <- resp
	}()
}

func (r *Raft) HandleVoteRequest(receiverId int, request kvgo.VoteRequest) {
	go func() {
		url := fmt.Sprintf("http://%s/voteRequest", r.cfg.Cluster[receiverId])
		requestBody, err := json.Marshal(&request)
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

		var resp kvgo.VoteResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			slog.Error("can't unmarshal VoteResponse body", "error", err)
			return
		}

		resp.Id = receiverId
		resp.RequestInTerm = request.Term
		r.voteResponses <- resp
	}()
}
