package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	kv_go "github.com/Boar-D-White-Foundation/kv-go"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
)

func (r *Raft) StartHttpServer() {
	router := gin.Default()
	router.POST("/appendEntryRequest", r.postAppendEntryRequest)
	router.POST("/voteRequest", r.postVoteRequest)

	err := router.Run(r.cfg.Cluster[r.cfg.CurrentId])
	if err != nil {
		log.Println(fmt.Sprintf("[ERROR] router finished with error: %s"), err)
	}
}

func (r *Raft) postAppendEntryRequest(c *gin.Context) {
	var request kv_go.AppendEntryRequest
	if err := c.BindJSON(&request); err != nil {
		log.Println(fmt.Sprintf("[ERROR] can't read request body: %s", err))
	}

	message := appendEntryMessage{
		request:  request,
		response: make(chan kv_go.AppendEntryResponse),
	}

	r.appendEntries <- message
	response := <-message.response
	c.IndentedJSON(http.StatusOK, &response)
}

func (r *Raft) postVoteRequest(c *gin.Context) {
	var request kv_go.VoteRequest
	if err := c.BindJSON(&request); err != nil {
		log.Println(fmt.Sprintf("[ERROR] can't read request body: %s", err))
	}

	message := voteMessage{
		request:  request,
		response: make(chan kv_go.VoteResponse),
	}

	r.votes <- message
	response := <-message.response
	c.IndentedJSON(http.StatusOK, &response)
}

func (r *Raft) HandleAppendEntryRequest(receiverId int, request kv_go.AppendEntryRequest) {
	go func() {
		url := fmt.Sprintf("http://%s/appendEntryRequest", r.cfg.Cluster[receiverId])
		requestBody, err := json.Marshal(&request)
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't marshal AppendEntryRequest: %s", err))
			return
		}
		response, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't send AppendEntryRequest: %s", err))
			return
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't read AppendEntryResponse body: %s", err))
			return
		}

		var resp kv_go.AppendEntryResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't unmarshal AppendEntryResponse body: %s", err))
			return
		}

		r.appendEntryResponses <- resp
	}()
}

func (r *Raft) HandleVoteRequest(receiverId int, request kv_go.VoteRequest) {
	go func() {
		url := fmt.Sprintf("http://%s/voteRequest", r.cfg.Cluster[receiverId])
		requestBody, err := json.Marshal(&request)
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't marshal VoteRequest: %s", err))
			return
		}
		response, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't send VoteRequest: %s", err))
			return
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't read VoteResponse body: %s", err))
			return
		}

		var resp kv_go.VoteResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			log.Println(fmt.Errorf("[ERROR] can't unmarshal VoteResponse body: %s", err))
			return
		}

		resp.Id = receiverId
		resp.RequestInTerm = request.Term
		r.voteResponses <- resp
	}()
}
