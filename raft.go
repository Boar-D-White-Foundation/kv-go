package kv_go

import (
	"fmt"
	"math/rand"
	"time"
)

type Config struct {
	ElectionTimeoutMax int
	ElectionTimeoutMin int
	LeaderHeartbeat    int
	CurrentId          int
	Cluster            map[int]string
}

type Follower struct {
	cfg      *Config
	term     uint64
	votedFor int
}

type Candidate struct {
	cfg          *Config
	term         uint64
	maxTerm      uint64
	votes        []bool
	grantedVotes int
}

type Leader struct {
	cfg      *Config
	term     uint64
	votedFor int
}

type AppendEntryRequest struct {
	Id   int    `json:"id"`
	Term uint64 `json:"term"`
}

type AppendEntryResponse struct {
	Term    uint64 `json:"term"`
	Success bool   `json:"success"`
}

type AppendEntryRequestHandler interface {
	HandleAppendEntryRequest(receiverId int, r AppendEntryRequest)
}

type VoteRequest struct {
	Term        uint64 `json:"term"`
	CandidateId int    `json:"candidateId"`
}

type VoteRequestHandler interface {
	HandleVoteRequest(receiverId int, r VoteRequest)
}

type VoteResponse struct {
	Id          int
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"voteGranted"`
}

func MakeFollower(cfg *Config, term uint64) Follower {
	return Follower{
		cfg:      cfg,
		term:     term,
		votedFor: cfg.CurrentId,
	}
}

func (f *Follower) Term() uint64 {
	return f.term
}

func (f *Follower) Timeout() time.Duration {
	return time.Duration(f.cfg.ElectionTimeoutMin+rand.Intn(f.cfg.ElectionTimeoutMax-f.cfg.ElectionTimeoutMin)) * time.Millisecond
}

func (f *Follower) HandleElectionTimeout(handler VoteRequestHandler) Candidate {
	c := MakeCandidate(f.cfg, f.term+1)
	c.sendPromotion(handler)
	return c
}

func (f *Follower) HandleAppendEntry(r AppendEntryRequest) AppendEntryResponse {
	if f.Term() > r.Term {
		return AppendEntryResponse{Term: f.Term(), Success: false}
	}

	f.term = r.Term
	return AppendEntryResponse{Term: f.Term(), Success: true}
}

func (f *Follower) HandleVote(r VoteRequest) VoteResponse {
	if r.Term <= f.Term() {
		return VoteResponse{
			Term:        f.Term(),
			VoteGranted: false,
		}
	}

	if f.votedFor != f.cfg.CurrentId {
		return VoteResponse{
			Term:        f.Term(),
			VoteGranted: false,
		}
	}

	f.votedFor = r.CandidateId
	return VoteResponse{
		Term:        f.Term(),
		VoteGranted: true,
	}
}

func MakeCandidate(cfg *Config, term uint64) Candidate {
	votes := make([]bool, len(cfg.Cluster))
	c := Candidate{
		term:    term,
		maxTerm: term,
		cfg:     cfg,
		votes:   votes,
	}
	c.addVote(cfg.CurrentId)
	return c
}

func (c *Candidate) Term() uint64 {
	return c.term
}

func (c *Candidate) Timeout() time.Duration {
	return time.Duration(c.cfg.ElectionTimeoutMin+rand.Intn(c.cfg.ElectionTimeoutMax-c.cfg.ElectionTimeoutMin)) * time.Millisecond
}

func (c *Candidate) addVote(id int) {
	if !c.votes[id] {
		c.votes[id] = true
		c.grantedVotes++
	}
}

func (c *Candidate) sendPromotion(handler VoteRequestHandler) {
	for i := 0; i < len(c.cfg.Cluster); i++ {
		if i == c.cfg.CurrentId {
			continue
		}
		handler.HandleVoteRequest(i, VoteRequest{CandidateId: c.cfg.CurrentId, Term: c.Term()})
	}
}

func (c *Candidate) HandleElectionTimeout(handler VoteRequestHandler) Candidate {
	newC := MakeCandidate(c.cfg, c.maxTerm+1)
	newC.sendPromotion(handler)
	return newC
}

func (c *Candidate) HandleVote(r VoteRequest) (*Follower, VoteResponse) {
	if c.Term() < r.Term {
		f := MakeFollower(c.cfg, r.Term)
		return &f, VoteResponse{
			Term:        c.Term(),
			VoteGranted: true,
		}
	}
	return nil, VoteResponse{
		Term:        c.Term(),
		VoteGranted: false,
	}
}

func (c *Candidate) HandleVoteResponse(r VoteResponse, handler AppendEntryRequestHandler) (*Leader, error) {
	if r.VoteGranted {
		c.addVote(r.Id)
		if c.grantedVotes > len(c.cfg.Cluster)/2 {
			l := MakeLeader(c.cfg, c.term)
			l.SendHeartbeat(handler)
			return &l, nil
		}

		return nil, fmt.Errorf(fmt.Sprintf("can't reach magority [%v/%v]", c.grantedVotes, len(c.cfg.Cluster)))
	}

	if r.Term > c.maxTerm {
		c.maxTerm = r.Term
	}
	return nil, fmt.Errorf("vote not granted")
}

func (c *Candidate) HandleAppendEntry(entry AppendEntryRequest) (*Follower, AppendEntryResponse) {
	if c.Term() <= entry.Term {
		f := MakeFollower(c.cfg, entry.Term)
		return &f, AppendEntryResponse{Term: c.Term(), Success: true}
	}

	return nil, AppendEntryResponse{Term: c.Term(), Success: false}
}

func MakeLeader(cfg *Config, term uint64) Leader {
	return Leader{
		cfg:  cfg,
		term: term,
	}
}

func (l *Leader) Term() uint64 {
	return l.term
}

func (l *Leader) Timeout() time.Duration {
	return time.Duration(l.cfg.LeaderHeartbeat) * time.Millisecond
}

func (l *Leader) HandleAppendEntry(entry AppendEntryRequest) (*Follower, AppendEntryResponse) {
	if l.Term() == entry.Term {
		return nil, AppendEntryResponse{Term: l.Term(), Success: false}
	}
	if l.Term() > entry.Term {
		return nil, AppendEntryResponse{Term: l.Term(), Success: false}
	}

	f := MakeFollower(l.cfg, entry.Term)
	return &f, AppendEntryResponse{Term: l.Term(), Success: true}
}

func (l *Leader) HandleVote(r VoteRequest) VoteResponse {
	if r.Term <= l.Term() {
		return VoteResponse{
			Term:        l.Term(),
			VoteGranted: false,
		}
	}

	if l.votedFor != l.cfg.CurrentId {
		return VoteResponse{
			Term:        l.Term(),
			VoteGranted: false,
		}
	}

	l.votedFor = r.CandidateId
	return VoteResponse{
		Term:        l.Term(),
		VoteGranted: true,
	}
}

func (l *Leader) SendHeartbeat(handler AppendEntryRequestHandler) {
	for i := 0; i < len(l.cfg.Cluster); i++ {
		if i == l.cfg.CurrentId {
			continue
		}
		handler.HandleAppendEntryRequest(i, AppendEntryRequest{Id: l.cfg.CurrentId, Term: l.term})
	}
}
