package kv_go

import (
	"testing"
)

// Leaders send periodic heartbeats (AppendEntries RPCs that carry no log entries) to all followers
// in order to maintain their authority.
// If a follower receives no communication over a period of time called the election timeout,
// then it assumes there is no viable leader and begins an election to choose a new leader.
// To begin an election, a follower increments its current term and transitions to candidate state.
func TestFollower_BecameCandidateIfNoLeadersHeartbeat(t *testing.T) {
	f := MakeFollower(makeConfig(5), 0)
	c := f.HandleElectionTimeout(makeVoteRequestHandlerMock())

	if f.Term()+1 != c.Term() {
		t.Fatalf("Follower should increment its term")
	}
}

func TestFollower_UpdateCurrentTermOnAppendEntryRequest(t *testing.T) {
	f := MakeFollower(makeConfig(5), 1)

	if resp := f.HandleAppendEntry(AppendEntryRequest{Term: 0}); resp.Success {
		t.Fatalf("Follower can't accept smaller term")
	}

	entry := AppendEntryRequest{Term: 1}
	_ = f.HandleAppendEntry(entry)
	if f.Term() != entry.Term {
		t.Fatalf("Follower should accept higher term")
	}
}

// The third possible outcome is that a candidate neither wins nor loses the election:
// if many followers become candidates at the same time, votes could be split so that no candidate obtains a majority.
// When this happens, each candidate will time out and start a new election by incrementing its term
// and initiating another round of RequestVote RPCs.
func TestCandidate_StartNewElectionAfterTimeout(t *testing.T) {
	c := MakeCandidate(makeConfig(5), 0)
	cNew := c.HandleElectionTimeout(makeVoteRequestHandlerMock())
	if c.Term()+1 != cNew.Term() {
		t.Fatalf("Candidate should increment its term before starting new election round")
	}
}

// A candidate starts new election using max term value getting from response vote
func TestCandidate_StartNewElectionWithHighestTerm(t *testing.T) {
	c := MakeCandidate(makeConfig(5), 0)
	vote := VoteResponse{Id: 1, Term: 1, VoteGranted: false}
	_, _ = c.HandleVoteResponse(vote, makeAppendRequestHandlerMock())

	cNew := c.HandleElectionTimeout(makeVoteRequestHandlerMock())
	if vote.Term+1 != cNew.Term() {
		t.Fatalf("Candidate should use higest term from VoteResponse")
	}
}

// A candidate wins an election if it receives votes from a majority of the servers in the full cluster for the same term.
// Each server will vote for at most one candidate in a given term, on a first-come-first-served basis
func TestCandidate_BecameLeaderAfterGettingMajorityVotes(t *testing.T) {
	c := MakeCandidate(makeConfig(5), 0)
	if l, _ := c.HandleVoteResponse(VoteResponse{Id: 1, VoteGranted: true}, makeAppendRequestHandlerMock()); l != nil {
		t.Fatalf("Candidate can't became a leader before getting mejority votes")
	}
	if l, _ := c.HandleVoteResponse(VoteResponse{Id: 2, VoteGranted: true}, makeAppendRequestHandlerMock()); l == nil {
		t.Fatalf("Candidate should became a leader after getting mejority votes")
	}
}

// Upon election: leader send initial empty AppendEntries RPCs (heartbeat) to each server
func TestCandidate_SendHeartbeatAfterBecameLeader(t *testing.T) {
	c := MakeCandidate(makeConfig(3), 0)
	mock := makeAppendRequestHandlerMock()
	_, _ = c.HandleVoteResponse(VoteResponse{Id: 1, VoteGranted: true}, mock)
	if len(mock.m) != 2 {
		t.Fatalf("Leader should send heatbeat to all his followers")
	}
}

// While waiting for votes, a candidate may receive an AppendEntries RPC from another server claiming to be leader.
// If the leader’s term (included in its RPC) is at least as large as the candidate’s current term,
// then the candidate recognizes the leader as legitimate and returns to follower state.
// If the term in the RPC is smaller than the candidate’s current term,
// then the candidate rejects the RPC and continues in candidate state.
func TestCandidate_BecameFollowerWhenLeaderAppeared(t *testing.T) {
	c := MakeCandidate(makeConfig(5), 1)
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
	l := MakeLeader(makeConfig(5), 1)
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

type appendRequestHandlerMock struct {
	m map[int]AppendEntryRequest
}

func makeAppendRequestHandlerMock() *appendRequestHandlerMock {
	return &appendRequestHandlerMock{
		m: make(map[int]AppendEntryRequest),
	}
}

func (mock *appendRequestHandlerMock) HandleAppendEntryRequest(receiverId int, r AppendEntryRequest) {
	mock.m[receiverId] = r
}

type voteRequestHandlerMock struct {
	m map[int]VoteRequest
}

func makeVoteRequestHandlerMock() *voteRequestHandlerMock {
	return &voteRequestHandlerMock{
		m: make(map[int]VoteRequest),
	}
}

func (mock *voteRequestHandlerMock) HandleVoteRequest(receiverId int, r VoteRequest) {
	mock.m[receiverId] = r
}
