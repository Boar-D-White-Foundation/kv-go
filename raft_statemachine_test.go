package kvgo

import (
	"testing"
	"time"
)

func TestRaft_FromCandidateToLeaderHappyPath(t *testing.T) {
	cfg := makeConfig(3)

	timer := &timerMock{
		t: make(chan time.Time),
	}
	handler := &requestHandleOnChannelsMock{
		ae: make(chan AppendEntryRequest, 2),
		v:  make(chan VoteRequest, 2),
	}

	raft := MakeRaft(*cfg, timer)
	raft.SetHandler(handler)
	stopped := make(chan bool)
	go func() {
		_ = raft.Start()
		stopped <- true
	}()

	// trigger election timeout and waiting VoteRequests
	timer.t <- time.Now()
	for i := 0; i < len(cfg.Cluster)-1; i++ {
		req := <-handler.v
		raft.SendVoteResponse(VoteResponse{
			ReceiverId:    req.ReceiverId,
			RequestInTerm: req.Term,
			Term:          req.Term,
			VoteGranted:   true,
		})
	}

	// waiting leader heartbeat
	for i := 0; i < len(cfg.Cluster)-1; i++ {
		req := <-handler.ae
		raft.SendAppendEntryResponse(AppendEntryResponse{
			Term:    req.Term,
			Success: true,
		})
	}

	raft.Stop()
	<-stopped
}

type requestHandleOnChannelsMock struct {
	ae chan AppendEntryRequest
	v  chan VoteRequest
}

func (r *requestHandleOnChannelsMock) HandleAppendEntryRequest(req AppendEntryRequest) {
	r.ae <- req
}

func (r *requestHandleOnChannelsMock) HandleVoteRequest(req VoteRequest) {
	r.v <- req
}

type timerMock struct {
	t chan time.Time
}

func (t *timerMock) Reset(from time.Duration, to time.Duration) {
}

func (t *timerMock) Chan() <-chan time.Time {
	return t.t
}
