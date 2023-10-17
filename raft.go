package kvgo

import (
	"github.com/Boar-D-White-Foundation/kvgo/internal/rtimer"
	"sort"
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
	cfg         *Config
	term        uint64
	log         []LogEntry
	commitIndex int
	votedFor    int
}

type Candidate struct {
	cfg          *Config
	term         uint64
	log          []LogEntry
	commitIndex  int
	votes        []bool
	grantedVotes int
}

type Leader struct {
	cfg         *Config
	term        uint64
	log         []LogEntry
	commitIndex int
	nextIndex   []int
	matchIndex  []int
}

type Command struct {
	Value int
}

type LogEntry struct {
	Term    uint64
	Command Command
}

type AppendEntryRequest struct {
	Id           int
	ReceiverId   int
	Term         uint64
	PrevLogTerm  uint64
	PrevLogIndex int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntryResponse struct {
	ReceiverId    int
	RequestInTerm uint64
	MatchIndex    int
	Term          uint64
	Success       bool
}

type VoteRequest struct {
	ReceiverId   int
	Term         uint64
	CandidateId  int
	PrevLogTerm  uint64
	PrevLogIndex int
}

type VoteResponse struct {
	ReceiverId    int
	RequestInTerm uint64
	Term          uint64
	VoteGranted   bool
}

type RequestHandler interface {
	HandleAppendEntryRequest(r AppendEntryRequest)
	HandleVoteRequest(r VoteRequest)
}

func MakeFollower(cfg *Config, term uint64, log []LogEntry, commitIndex int) Follower {
	return Follower{
		cfg:         cfg,
		term:        term,
		votedFor:    cfg.CurrentId,
		log:         log,
		commitIndex: commitIndex,
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
	c := MakeCandidate(f.cfg, f.term+1, f.log, f.commitIndex)
	c.sendPromotion(handler)
	return c
}

func (f *Follower) HandleAppendEntry(r AppendEntryRequest) (*Follower, AppendEntryResponse) {
	if f.Term() > r.Term {
		return nil, AppendEntryResponse{Term: f.Term(), Success: false}
	}
	if f.Term() == r.Term {
		return nil, f.appendToEntryLog(r)
	}

	newF := MakeFollower(f.cfg, r.Term, f.log, f.commitIndex)
	return &newF, newF.appendToEntryLog(r)
}

func (f *Follower) appendToEntryLog(r AppendEntryRequest) AppendEntryResponse {
	if r.PrevLogIndex >= len(f.log) {
		return AppendEntryResponse{
			Term:    f.Term(),
			Success: false,
		}
	}
	if f.log[r.PrevLogIndex].Term != r.PrevLogTerm {
		return AppendEntryResponse{
			Term:    f.Term(),
			Success: false,
		}
	}

	if len(r.Entries) != 0 {
		f.log = append(f.log, r.Entries...)
	}
	if r.LeaderCommit > f.commitIndex {
		if r.LeaderCommit >= len(f.log) {
			f.commitIndex = len(f.log) - 1
		} else {
			f.commitIndex = r.LeaderCommit
		}
	}
	return AppendEntryResponse{
		Term:    f.Term(),
		Success: true,
	}
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

	voteGranted := true
	if r.PrevLogTerm < f.log[len(f.log)-1].Term {
		voteGranted = false
	}
	if r.PrevLogTerm == f.log[len(f.log)-1].Term && r.PrevLogIndex < len(f.log)-1 {
		voteGranted = false
	}

	newF := MakeFollower(f.cfg, r.Term, f.log, f.commitIndex)
	newF.votedFor = r.CandidateId
	return &newF, VoteResponse{
		Term:        newF.Term(),
		VoteGranted: voteGranted,
	}
}

func MakeCandidate(cfg *Config, term uint64, log []LogEntry, commitIndex int) Candidate {
	votes := make([]bool, len(cfg.Cluster))
	c := Candidate{
		term:        term,
		log:         log,
		commitIndex: commitIndex,
		cfg:         cfg,
		votes:       votes,
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
		handler.HandleVoteRequest(VoteRequest{
			ReceiverId:   i,
			CandidateId:  c.cfg.CurrentId,
			Term:         c.Term(),
			PrevLogIndex: len(c.log) - 1,
			PrevLogTerm:  c.log[len(c.log)-1].Term,
		})
	}
}

func (c *Candidate) HandleElectionTimeout(handler RequestHandler) Candidate {
	newC := MakeCandidate(c.cfg, c.term+1, c.log, c.commitIndex)
	newC.sendPromotion(handler)
	return newC
}

func (c *Candidate) HandleVote(r VoteRequest) (*Follower, VoteResponse) {
	if c.Term() > r.Term {
		return nil, VoteResponse{
			Term:        c.Term(),
			VoteGranted: false,
		}
	}

	voteGranted := true
	if r.PrevLogTerm < c.log[len(c.log)-1].Term {
		voteGranted = false
	}
	if r.PrevLogTerm == c.log[len(c.log)-1].Term && r.PrevLogIndex < len(c.log)-1 {
		voteGranted = false
	}

	if c.Term() == r.Term && !voteGranted {
		return nil, VoteResponse{
			Term:        c.Term(),
			VoteGranted: false,
		}
	}

	f := MakeFollower(c.cfg, r.Term, c.log, c.commitIndex)
	f.votedFor = r.CandidateId
	return &f, VoteResponse{
		Term:        f.Term(),
		VoteGranted: voteGranted,
	}

}

func (c *Candidate) HandleVoteResponse(r VoteResponse, handler RequestHandler) (*Leader, *Follower) {
	if c.Term() == r.RequestInTerm && r.VoteGranted {
		c.addVote(r.ReceiverId)
		if c.grantedVotes > len(c.cfg.Cluster)/2 {
			l := MakeLeader(c.cfg, c.term, c.log, c.commitIndex)
			l.SendHeartbeat(handler)
			return &l, nil
		}

		return nil, nil
	}

	if c.Term() < r.Term {
		f := MakeFollower(c.cfg, r.Term, c.log, c.commitIndex)
		return nil, &f
	}
	return nil, nil
}

func (c *Candidate) HandleAppendEntry(r AppendEntryRequest) (*Follower, AppendEntryResponse) {
	if c.Term() <= r.Term {
		f := MakeFollower(c.cfg, r.Term, c.log, c.commitIndex)
		return &f, f.appendToEntryLog(r)
	}

	return nil, AppendEntryResponse{Term: c.Term(), Success: false}
}

func MakeLeader(cfg *Config, term uint64, log []LogEntry, commitIndex int) Leader {
	nextIndex := make([]int, len(cfg.Cluster))
	for i, _ := range nextIndex {
		nextIndex[i] = len(log)
	}
	return Leader{
		cfg:         cfg,
		term:        term,
		log:         log,
		commitIndex: commitIndex,
		nextIndex:   nextIndex,
		matchIndex:  make([]int, len(cfg.Cluster)),
	}
}

func (l *Leader) Term() uint64 {
	return l.term
}

func (l *Leader) SetTimeout(timer rtimer.Timer) {
	timer.Reset(l.cfg.LeaderHeartbeat, l.cfg.LeaderHeartbeat)
}

func (l *Leader) HandleAppendEntry(r AppendEntryRequest) (*Follower, AppendEntryResponse) {
	if l.Term() == r.Term {
		return nil, AppendEntryResponse{Term: l.Term(), Success: false}
	}
	if l.Term() > r.Term {
		return nil, AppendEntryResponse{Term: l.Term(), Success: false}
	}

	f := MakeFollower(l.cfg, r.Term, l.log, l.commitIndex)
	return &f, f.appendToEntryLog(r)
}

func (l *Leader) HandleVote(r VoteRequest) (*Follower, VoteResponse) {
	if r.Term <= l.Term() {
		return nil, VoteResponse{
			Term:        l.Term(),
			VoteGranted: false,
		}
	}

	voteGranted := true
	if r.PrevLogTerm < l.log[len(l.log)-1].Term {
		voteGranted = false
	}
	if r.PrevLogTerm == l.log[len(l.log)-1].Term && r.PrevLogIndex < len(l.log)-1 {
		voteGranted = false
	}

	f := MakeFollower(l.cfg, r.Term, l.log, l.commitIndex)
	f.votedFor = r.CandidateId
	return &f, VoteResponse{
		Term:        l.Term(),
		VoteGranted: voteGranted,
	}
}

func (l *Leader) SendHeartbeat(handler RequestHandler) {
	for i := 0; i < len(l.cfg.Cluster); i++ {
		if i == l.cfg.CurrentId {
			continue
		}

		entries := make([]LogEntry, 0, 1)
		if l.nextIndex[i] < len(l.log) {
			entries = append(entries, l.log[l.nextIndex[i]])
		}
		request := AppendEntryRequest{
			ReceiverId:   i,
			Id:           l.cfg.CurrentId,
			Term:         l.term,
			Entries:      entries,
			LeaderCommit: l.commitIndex,
			PrevLogIndex: l.nextIndex[i] - 1,
			PrevLogTerm:  l.log[l.nextIndex[i]-1].Term,
		}
		handler.HandleAppendEntryRequest(request)
	}
}

func (l *Leader) ExecuteCommand(cmd Command) {
	l.log = append(l.log, LogEntry{Term: l.Term(), Command: cmd})
}

func (l *Leader) HandleAppendEntryResponse(r AppendEntryResponse) *Follower {
	if l.Term() != r.RequestInTerm {
		return nil
	}
	if l.Term() < r.Term {
		f := MakeFollower(l.cfg, r.Term, l.log, l.commitIndex)
		return &f
	}
	if r.Success {
		l.matchIndex[r.ReceiverId] = r.MatchIndex
		l.nextIndex[r.ReceiverId] = r.MatchIndex + 1
		l.updateCommitIndex()
	} else {
		l.nextIndex[r.ReceiverId]--
	}

	return nil
}

func (l *Leader) updateCommitIndex() {
	matches := make([]int, len(l.cfg.Cluster))
	for i, _ := range matches {
		if i == l.cfg.CurrentId {
			matches[i] = len(l.log) - 1
			continue
		}
		matches[i] = l.matchIndex[i]
	}

	sort.Ints(matches)
	majorityIndex := matches[len(matches)/2]
	if majorityIndex > l.commitIndex {
		l.commitIndex = majorityIndex
	}
}
