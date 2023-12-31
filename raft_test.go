package kvgo

import (
	"testing"
)

// Leaders send periodic heartbeats (AppendEntries RPCs that carry no log entries) to all followers
// in order to maintain their authority.
// If a follower receives no communication over a period of time called the election timeout,
// then it assumes there is no viable leader and begins an election to choose a new leader.
// To begin an election, a follower increments its current term and transitions to candidate state.
func TestFollower_BecameCandidateIfNoLeadersHeartbeat(t *testing.T) {
	f := makeFollowerForTest(5, 0)
	c := f.HandleElectionTimeout(makeRequestHandlerMock())

	if f.Term()+1 != c.Term() {
		t.Fatalf("Follower should increment its term")
	}
}

func TestFollower_RejectVoteRequestWithSmallerTerm(t *testing.T) {
	f := makeFollowerForTest(5, 1)
	newF, resp := f.HandleVote(VoteRequest{Term: 0, CandidateId: 1})
	if resp.VoteGranted {
		t.Fatalf("Follower should reject Request with smaller term")
	}
	if resp.Term != f.Term() {
		t.Fatalf("Follower should reply with his term")
	}
	if newF != nil {
		t.Fatalf("Follower should not change his state")
	}
}

func TestFollower_RejectVoteRequestWithEqualTerm(t *testing.T) {
	f := makeFollowerForTest(5, 1)
	newF, resp := f.HandleVote(VoteRequest{Term: 1, CandidateId: 1})
	if resp.VoteGranted {
		t.Fatalf("Follower should reject Request with equal term")
	}
	if resp.Term != f.Term() {
		t.Fatalf("Follower should reply with his term")
	}
	if newF != nil {
		t.Fatalf("Follower should not change his state")
	}
}

func TestFollower_AcceptVoteRequestWithHigherTerm(t *testing.T) {
	f := makeFollowerForTest(5, 1)
	req := VoteRequest{Term: 2, CandidateId: 1}
	newF, resp := f.HandleVote(req)
	if !resp.VoteGranted {
		t.Fatalf("Follower should accept Request with higher term")
	}
	if newF == nil || newF.term != req.Term {
		t.Fatalf("Follower should update his term from requset")
	}
	if newF.votedFor != req.CandidateId {
		t.Fatalf("Follower should safe his vote")
	}
	if resp.Term != newF.Term() {
		t.Fatalf("Follower should reply with his term")
	}
}

func TestFollower_RejectAppendEntryRequestWithSmallerTerm(t *testing.T) {
	f := makeFollowerForTest(5, 1)

	newF, resp := f.HandleAppendEntry(AppendEntryRequest{Term: 0})
	if resp.Success {
		t.Fatalf("Follower should reject requset with equal term")
	}
	if newF != nil {
		t.Fatalf("Follower should not change his state")
	}
}

func TestFollower_AcceptAppendEntryWithEqualTerm(t *testing.T) {
	f := makeFollowerForTest(5, 1)

	newF, resp := f.HandleAppendEntry(AppendEntryRequest{Term: 1})
	if !resp.Success {
		t.Fatalf("Follower should accept requset with equal term")
	}
	if newF != nil {
		t.Fatalf("Follower should not update his state if accept euqal term")
	}
}

func TestFollower_AcceptAppendEntryWithHigherTerm(t *testing.T) {
	f := makeFollowerForTest(5, 1)

	req := AppendEntryRequest{Term: 2}
	newF, resp := f.HandleAppendEntry(req)
	if !resp.Success {
		t.Fatalf("Follower should accept requset with equal term")
	}
	if newF == nil || newF.term != req.Term {
		t.Fatalf("Follower should update his term from requset")
	}
}

// The third possible outcome is that a candidate neither wins nor loses the election:
// if many followers become candidates at the same time, votes could be split so that no candidate obtains a majority.
// When this happens, each candidate will time out and start a new election by incrementing its term
// and initiating another round of RequestVote RPCs.
func TestCandidate_StartNewElectionAfterTimeout(t *testing.T) {
	c := makeCandidateForTest(5, 0)
	cNew := c.HandleElectionTimeout(makeRequestHandlerMock())
	if c.Term()+1 != cNew.Term() {
		t.Fatalf("Candidate should increment its term before starting new election round")
	}
}

func TestCandidate_BecameFollowerWhenVoteResponseContainHigherTerm(t *testing.T) {
	c := makeCandidateForTest(5, 0)
	vote := VoteResponse{ReceiverId: 1, Term: 1, VoteGranted: false}
	_, f := c.HandleVoteResponse(vote, makeRequestHandlerMock())
	if f == nil {
		t.Fatalf("Candidate should became a follower after getting Response with higer term")
	}
	if f.Term() != vote.Term {
		t.Fatalf("Follower should get his term from Request")
	}
}

func TestCandidate_IgnoreVoteRequestWithSmallerTerm(t *testing.T) {
	c := makeCandidateForTest(5, 1)
	req := VoteRequest{Term: 0, CandidateId: 1}
	f, resp := c.HandleVote(req)

	if f != nil {
		t.Fatalf("Candidate should not change his state")
	}
	if resp.VoteGranted {
		t.Fatalf("Candidate should not granted vote")
	}
	if resp.Term != c.Term() {
		t.Fatalf("VoteResponse should has candidates' term")
	}
}

func TestCandidate_BecameFollowerWhenVoteRequestContainHigherTerm(t *testing.T) {
	c := makeCandidateForTest(5, 0)
	req := VoteRequest{Term: 1, CandidateId: 1}
	f, resp := c.HandleVote(req)
	if f == nil {
		t.Fatalf("Candidate should became a follower after getting requset with higer term")
	}
	if f.Term() != req.Term {
		t.Fatalf("Follower should get his term from Request")
	}
	if !resp.VoteGranted {
		t.Fatalf("Candidate should grant his vote to candidate with higher term")
	}
	if f.votedFor != req.CandidateId {
		t.Fatalf("Candidate should save votedFor in next state")
	}
}

func TestCandidate_IgnoreVoteResponseFromPreviousTerm(t *testing.T) {
	c := makeCandidateForTest(5, 2)
	vote := VoteResponse{ReceiverId: 1, RequestInTerm: 1, Term: 1, VoteGranted: true}
	_, _ = c.HandleVoteResponse(vote, makeRequestHandlerMock())
	if c.grantedVotes != 1 {
		t.Fatalf("Candidate should ingore Response from previous term")
	}
}

// A candidate wins an election if it receives votes from a majority of the servers in the full cluster for the same term.
// Each server will vote for at most one candidate in a given term, on a first-come-first-served basis
func TestCandidate_BecameLeaderAfterGettingMajorityVotes(t *testing.T) {
	c := makeCandidateForTest(5, 0)
	if l, _ := c.HandleVoteResponse(VoteResponse{ReceiverId: 1, VoteGranted: true}, makeRequestHandlerMock()); l != nil {
		t.Fatalf("Candidate can't became a leader before getting mejority votes")
	}
	if l, _ := c.HandleVoteResponse(VoteResponse{ReceiverId: 2, VoteGranted: true}, makeRequestHandlerMock()); l == nil {
		t.Fatalf("Candidate should became a leader after getting mejority votes")
	}
}

// Upon election: leader send initial empty AppendEntries RPCs (heartbeat) to each server
func TestCandidate_SendHeartbeatAfterBecameLeader(t *testing.T) {
	c := makeCandidateForTest(3, 0)
	mock := makeRequestHandlerMock()
	_, _ = c.HandleVoteResponse(VoteResponse{ReceiverId: 1, VoteGranted: true}, mock)
	if len(mock.mAppendEntries) != 2 {
		t.Fatalf("Leader should send heatbeat to all his followers")
	}
}

// While waiting for votes, a candidate may receive an AppendEntries RPC from another server claiming to be leader.
// If the leader’s term (included in its RPC) is at least as large as the candidate’s current term,
// then the candidate recognizes the leader as legitimate and returns to follower state.
// If the term in the RPC is smaller than the candidate’s current term,
// then the candidate rejects the RPC and continues in candidate state.
func TestCandidate_BecameFollowerWhenLeaderAppeared(t *testing.T) {
	c := makeCandidateForTest(5, 1)
	if f, _ := c.HandleAppendEntry(AppendEntryRequest{Term: 0}); f != nil {
		t.Fatalf("Candidate can't became a follower after getting smaller term")
	}
	if f, _ := c.HandleAppendEntry(AppendEntryRequest{Term: 1}); f == nil {
		t.Fatalf("Candidate should became a follower after getting equal term")
	}
	if f, _ := c.HandleAppendEntry(AppendEntryRequest{Term: 2}); f == nil {
		t.Fatalf("Candidate should became a follower after getting greater term")
	}
}

func TestLeader_BecameFollowerWhenAnotherLeaderAppeared(t *testing.T) {
	l := makeLeaderForTest(5, 1)
	if f, _ := l.HandleAppendEntry(AppendEntryRequest{Term: 0}); f != nil {
		t.Fatalf("Leader can't became a follower after getting smaller term")
	}
	if f, _ := l.HandleAppendEntry(AppendEntryRequest{Term: 1}); f != nil {
		t.Fatalf("Leader can't became a follower after getting equal term")
	}
	if f, _ := l.HandleAppendEntry(AppendEntryRequest{Term: 2}); f == nil {
		t.Fatalf("Leader should became a follower after getting greater term")
	}
}

func TestLeader_RejectVoteRequestWithSmallerTerm(t *testing.T) {
	l := makeLeaderForTest(5, 1)
	f, resp := l.HandleVote(VoteRequest{Term: 0, CandidateId: 1})
	if resp.VoteGranted {
		t.Fatalf("Leader should reject Request with smaller term")
	}
	if resp.Term != l.Term() {
		t.Fatalf("Leader should reply with his term")
	}
	if f != nil {
		t.Fatalf("Leader should not change his state")
	}
}

func TestLeader_RejectVoteRequestWithEqualTerm(t *testing.T) {
	l := makeLeaderForTest(5, 1)
	f, resp := l.HandleVote(VoteRequest{Term: 1, CandidateId: 1})
	if resp.VoteGranted {
		t.Fatalf("Follower should reject Request with equal term")
	}
	if resp.Term != l.Term() {
		t.Fatalf("Follower should reply with his term")
	}
	if f != nil {
		t.Fatalf("Follower should not change his state")
	}
}

func TestLeader_BecameFollowerWhenAcceptVoteRequestWithHigherTerm(t *testing.T) {
	l := makeLeaderForTest(5, 1)
	req := VoteRequest{Term: 2, CandidateId: 1}
	f, resp := l.HandleVote(req)
	if !resp.VoteGranted {
		t.Fatalf("Leader should accept Request with higher term")
	}
	if f == nil || f.term != req.Term {
		t.Fatalf("Leader should update his term from requset")
	}
	if f.votedFor != req.CandidateId {
		t.Fatalf("Follower should safe his vote")
	}
	if resp.Term != l.Term() {
		t.Fatalf("Leader should reply with his term")
	}
}

func TestLeader_SendLogEntriesToFollowers(t *testing.T) {
	l := makeLeaderForTest(3, 1)
	cmd := Command{Value: 10}
	l.ExecuteCommand(cmd)

	mock := makeRequestHandlerMock()
	l.SendHeartbeat(mock)

	for _, v := range mock.mAppendEntries {
		if len(v.Entries) == 0 {
			t.Fatalf("Leader should send entries after getting command")
		}
		if len(v.Entries) == 0 || v.Entries[0].Command != cmd || v.Entries[0].Term != l.Term() {
			t.Fatalf("Entries contains invalid properties")
		}
	}
}

func TestLeader_IncreasingCommitIndexAfterReplicatingInMajority(t *testing.T) {
	l := makeLeaderForTest(3, 1)
	cmd := Command{Value: 10}
	l.ExecuteCommand(cmd)

	mock := makeRequestHandlerMock()
	l.SendHeartbeat(mock)
	for _, v := range mock.mAppendEntries {
		if v.LeaderCommit != 0 {
			t.Fatalf("Leader should not increase commit index before replicating in majority")
		}
	}

	l.HandleAppendEntryResponse(AppendEntryResponse{
		ReceiverId:    1,
		RequestInTerm: l.Term(),
		MatchIndex:    1,
		Term:          l.Term(),
		Success:       true,
	})

	l.SendHeartbeat(mock)
	for _, v := range mock.mAppendEntries {
		if v.LeaderCommit != 1 {
			t.Fatalf("Leader should update commit index after replicating in majority")
		}
	}
}

func TestLeader_DecreasePrevLogIndexIfFollowerDeclinedAppendEntry(t *testing.T) {
	prevLog := make([]LogEntry, 1)
	prevLog = append(prevLog, LogEntry{
		Term:    1,
		Command: Command{},
	})
	prevLog = append(prevLog, LogEntry{
		Term:    1,
		Command: Command{},
	})
	l := MakeLeader(makeConfig(3), 2, prevLog, 2)

	mock := makeRequestHandlerMock()
	l.SendHeartbeat(mock)
	for _, v := range mock.mAppendEntries {
		if v.PrevLogIndex != 2 {
			t.Fatalf("Leader should send heartbeat with last known log index")
		}
		if v.PrevLogTerm != 1 {
			t.Fatalf("Prev log term should obtained form log enties")
		}
	}

	l.HandleAppendEntryResponse(AppendEntryResponse{
		ReceiverId:    1,
		RequestInTerm: l.Term(),
		MatchIndex:    0,
		Term:          l.Term(),
		Success:       false,
	})
	l.SendHeartbeat(mock)

	if mock.mAppendEntries[1].PrevLogIndex != 1 {
		t.Fatalf("Leader should decrease prev log index after getting failed response")
	}
	if mock.mAppendEntries[2].PrevLogIndex != 2 {
		t.Fatalf("Prev log term should not change for other followers")
	}
}

func makeConfig(clusterSize int) *Config {
	cluster := make(map[int]string)
	for i := 0; i < clusterSize; i++ {
		cluster[i] = ""
	}

	return &Config{
		CurrentId: 0,
		Cluster:   cluster,
	}
}

type requestHandlerMock struct {
	mAppendEntries map[int]AppendEntryRequest
	mVotes         map[int]VoteRequest
}

func makeFollowerForTest(clusterSize int, term uint64) Follower {
	return MakeFollower(makeConfig(clusterSize), term, make([]LogEntry, 1), 0)
}

func makeCandidateForTest(clusterSize int, term uint64) Candidate {
	return MakeCandidate(makeConfig(clusterSize), term, make([]LogEntry, 1), 0)
}

func makeLeaderForTest(clusterSize int, term uint64) Leader {
	return MakeLeader(makeConfig(clusterSize), term, make([]LogEntry, 1), 0)
}

func makeRequestHandlerMock() *requestHandlerMock {
	return &requestHandlerMock{
		mAppendEntries: make(map[int]AppendEntryRequest),
		mVotes:         make(map[int]VoteRequest),
	}
}

func (mock *requestHandlerMock) HandleAppendEntryRequest(r AppendEntryRequest) {
	mock.mAppendEntries[r.ReceiverId] = r
}

func (mock *requestHandlerMock) HandleVoteRequest(r VoteRequest) {
	mock.mVotes[r.ReceiverId] = r
}
