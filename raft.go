package kvgo

import (
	"github.com/Boar-D-White-Foundation/kvgo/internal/rtimer"
	"time"
)

type Config struct {
	ElectionTimeoutMax time.Duration
	ElectionTimeoutMin time.Duration
	LeaderHeartbeat    time.Duration
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
	votes        []bool
	grantedVotes int
}

type Leader struct {
	cfg  *Config
	term uint64
}

type AppendEntryRequest struct {
	Id         int `json:"id"`
	ReceiverId int
	Term       uint64 `json:"term"`
}

type AppendEntryResponse struct {
	Term    uint64 `json:"term"`
	Success bool   `json:"success"`
}

type VoteRequest struct {
	ReceiverId  int
	Term        uint64 `json:"term"`
	CandidateId int    `json:"candidateId"`
}

type VoteResponse struct {
	Id            int
	RequestInTerm uint64
	Term          uint64 `json:"term"`
	VoteGranted   bool   `json:"voteGranted"`
}

type RequestHandler interface {
	HandleAppendEntryRequest(r AppendEntryRequest)
	HandleVoteRequest(r VoteRequest)
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

func (f *Follower) VoteFor() int {
	return f.votedFor
}

func (f *Follower) SetTimeout(timer rtimer.Timer) {
	timer.Reset(f.cfg.ElectionTimeoutMin, f.cfg.ElectionTimeoutMax)
}

func (f *Follower) HandleElectionTimeout(handler RequestHandler) Candidate {
	c := MakeCandidate(f.cfg, f.term+1)
	c.sendPromotion(handler)
	return c
}

func (f *Follower) HandleAppendEntry(r AppendEntryRequest) (*Follower, AppendEntryResponse) {
	if f.Term() > r.Term {
		return nil, AppendEntryResponse{Term: f.Term(), Success: false}
	}
	if f.Term() == r.Term {
		return nil, AppendEntryResponse{Term: f.Term(), Success: true}
	}

	newF := MakeFollower(f.cfg, r.Term)
	return &newF, AppendEntryResponse{Term: f.Term(), Success: true}
}

func (f *Follower) HandleVote(r VoteRequest) (*Follower, VoteResponse) {
	if f.Term() > r.Term {
		return nil, VoteResponse{
			Term:        f.Term(),
			VoteGranted: false,
		}
	}
	if f.Term() == r.Term {
		return nil, VoteResponse{
			Term:        f.Term(),
			VoteGranted: f.votedFor == r.CandidateId,
		}
	}

	newF := MakeFollower(f.cfg, r.Term)
	newF.votedFor = r.CandidateId
	return &newF, VoteResponse{
		Term:        newF.Term(),
		VoteGranted: true,
	}
}

func MakeCandidate(cfg *Config, term uint64) Candidate {
	votes := make([]bool, len(cfg.Cluster))
	c := Candidate{
		term:  term,
		cfg:   cfg,
		votes: votes,
	}
	c.addVote(cfg.CurrentId)
	return c
}

func (c *Candidate) Term() uint64 {
	return c.term
}

func (c *Candidate) SetTimeout(timer rtimer.Timer) {
	timer.Reset(c.cfg.ElectionTimeoutMin, c.cfg.ElectionTimeoutMax)
}

func (c *Candidate) addVote(id int) {
	if !c.votes[id] {
		c.votes[id] = true
		c.grantedVotes++
	}
}

func (c *Candidate) sendPromotion(handler RequestHandler) {
	for i := 0; i < len(c.cfg.Cluster); i++ {
		if i == c.cfg.CurrentId {
			continue
		}
		handler.HandleVoteRequest(VoteRequest{ReceiverId: i, CandidateId: c.cfg.CurrentId, Term: c.Term()})
	}
}

func (c *Candidate) HandleElectionTimeout(handler RequestHandler) Candidate {
	newC := MakeCandidate(c.cfg, c.term+1)
	newC.sendPromotion(handler)
	return newC
}

func (c *Candidate) HandleVote(r VoteRequest) (*Follower, VoteResponse) {
	if c.Term() < r.Term {
		f := MakeFollower(c.cfg, r.Term)
		f.votedFor = r.CandidateId
		return &f, VoteResponse{
			Term:        f.Term(),
			VoteGranted: true,
		}
	}
	return nil, VoteResponse{
		Term:        c.Term(),
		VoteGranted: false,
	}
}

func (c *Candidate) HandleVoteResponse(r VoteResponse, handler RequestHandler) (*Leader, *Follower) {
	if c.Term() == r.RequestInTerm && r.VoteGranted {
		c.addVote(r.Id)
		if c.grantedVotes > len(c.cfg.Cluster)/2 {
			l := MakeLeader(c.cfg, c.term)
			l.SendHeartbeat(handler)
			return &l, nil
		}

		return nil, nil
	}

	if c.Term() < r.Term {
		f := MakeFollower(c.cfg, r.Term)
		return nil, &f
	}
	return nil, nil
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

func (l *Leader) SetTimeout(timer rtimer.Timer) {
	timer.Reset(l.cfg.LeaderHeartbeat, l.cfg.LeaderHeartbeat)
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

func (l *Leader) HandleVote(r VoteRequest) (*Follower, VoteResponse) {
	if r.Term <= l.Term() {
		return nil, VoteResponse{
			Term:        l.Term(),
			VoteGranted: false,
		}
	}

	f := MakeFollower(l.cfg, r.Term)
	f.votedFor = r.CandidateId
	return &f, VoteResponse{
		Term:        l.Term(),
		VoteGranted: true,
	}
}

func (l *Leader) SendHeartbeat(handler RequestHandler) {
	for i := 0; i < len(l.cfg.Cluster); i++ {
		if i == l.cfg.CurrentId {
			continue
		}
		handler.HandleAppendEntryRequest(AppendEntryRequest{ReceiverId: i, Id: l.cfg.CurrentId, Term: l.term})
	}
}
