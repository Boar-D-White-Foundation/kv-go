package kvgo

import (
	"fmt"
	"github.com/Boar-D-White-Foundation/kvgo/internal/rtimer"
	"log/slog"
)

type AppendEntryMessage struct {
	Request  AppendEntryRequest
	Response chan AppendEntryResponse
}

type VoteMessage struct {
	Request  VoteRequest
	Response chan VoteResponse
}

type Raft struct {
	cfg                  Config
	handler              RequestHandler
	timer                rtimer.Timer
	appendEntries        chan AppendEntryMessage
	votes                chan VoteMessage
	appendEntryResponses chan AppendEntryResponse
	voteResponses        chan VoteResponse
	terminate            chan bool
}

func MakeRaft(cfg Config, timer rtimer.Timer) Raft {
	return Raft{
		cfg:                  cfg,
		timer:                timer,
		terminate:            make(chan bool),
		appendEntries:        make(chan AppendEntryMessage),
		votes:                make(chan VoteMessage),
		appendEntryResponses: make(chan AppendEntryResponse),
		voteResponses:        make(chan VoteResponse),
	}
}

func (r *Raft) SetHandler(handler RequestHandler) {
	r.handler = handler
}

func (r *Raft) SendAppendEntryMessage(message AppendEntryMessage) {
	r.appendEntries <- message
}

func (r *Raft) SendVoteMessage(message VoteMessage) {
	r.votes <- message
}

func (r *Raft) SendVoteResponse(response VoteResponse) {
	r.voteResponses <- response
}

func (r *Raft) SendAppendEntryResponse(response AppendEntryResponse) {
	r.appendEntryResponses <- response
}

type state struct {
	s interface{}
}

func beginState(cfg *Config, timer rtimer.Timer) state {
	f := MakeFollower(cfg, 0, make([]LogEntry, 1), 0)
	f.SetTimeout(timer)
	return state{
		s: &f,
	}
}

func fromCandidate(c *Candidate, timer rtimer.Timer) state {
	c.SetTimeout(timer)
	return state{
		s: c,
	}
}

func fromFollower(f *Follower, timer rtimer.Timer) state {
	f.SetTimeout(timer)
	return state{
		s: f,
	}
}

func fromLeader(l *Leader, timer rtimer.Timer) state {
	l.SetTimeout(timer)
	return state{
		s: l,
	}
}

func (r *Raft) Stop() {
	r.terminate <- true
}

func (r *Raft) Start() error {
	if r.handler == nil {
		return fmt.Errorf("handler was not set")
	}

	st := beginState(&r.cfg, r.timer)
	for {
		select {
		case <-r.terminate:
			slog.Info("main loop stopped")
			return nil
		case appendEntry := <-r.appendEntries:
			switch s := st.s.(type) {
			case *Follower:
				f, resp := s.HandleAppendEntry(appendEntry.Request)
				appendEntry.Response <- resp
				if f != nil {
					slog.Info("follow for new leader", "term", f.Term())
					st = fromFollower(f, r.timer)
				} else {
					// reset timeout
					st = fromFollower(s, r.timer)
				}
			case *Candidate:
				f, resp := s.HandleAppendEntry(appendEntry.Request)
				appendEntry.Response <- resp
				if f != nil {
					slog.Info("became follower from candidate state", "term", f.Term())
					st = fromFollower(f, r.timer)
				}
			case *Leader:
				f, resp := s.HandleAppendEntry(appendEntry.Request)
				appendEntry.Response <- resp
				if f != nil {
					slog.Info("became follower from leader state", "term", f.Term())
					st = fromFollower(f, r.timer)
				}
			default:
				return fmt.Errorf("unknown type in appendEntries section")
			}
		case vote := <-r.votes:
			switch s := st.s.(type) {
			case *Follower:
				f, resp := s.HandleVote(vote.Request)
				vote.Response <- resp
				if f != nil {
					slog.Info("follow for new leader", "term", f.Term())
					if resp.VoteGranted {
						slog.Info("granted vote for", "voteFor", f.VoteFor())
					}
					st = fromFollower(f, r.timer)
				}
			case *Candidate:
				f, resp := s.HandleVote(vote.Request)
				vote.Response <- resp
				if f != nil {
					slog.Info("became follower from candidate state", "term", f.Term())
					if resp.VoteGranted {
						slog.Info("granted vote for", "voteFor", f.VoteFor())
					}
					st = fromFollower(f, r.timer)
				}
			case *Leader:
				f, resp := s.HandleVote(vote.Request)
				vote.Response <- resp
				if f != nil {
					slog.Info("became follower from leader state", "term", f.Term())
					if resp.VoteGranted {
						slog.Info("granted vote for", "voteFor", f.VoteFor())
					}
					st = fromFollower(f, r.timer)
				}
			default:
				return fmt.Errorf("unknown type in votes section")
			}
		case voteResp := <-r.voteResponses:
			switch s := st.s.(type) {
			case *Follower:
				slog.Debug("skip vote Response for follower")
			case *Candidate:
				l, f := s.HandleVoteResponse(voteResp, r.handler)
				if f != nil {
					slog.Info("became follower from candidate state", "term", f.Term())
					st = fromFollower(f, r.timer)
				}
				if l != nil {
					slog.Info("became a leader", "term", l.Term())
					st = fromLeader(l, r.timer)
				}
			case *Leader:
				slog.Debug("skip vote Response for leader")
			default:
				return fmt.Errorf("unknown type in voteResponse section")
			}
		case appendEntryResp := <-r.appendEntryResponses:
			switch s := st.s.(type) {
			case *Follower:
				slog.Debug("skip append entry Response for follower")
			case *Candidate:
				slog.Debug("skip append entry Response for candidate")
			case *Leader:
				f := s.HandleAppendEntryResponse(appendEntryResp)
				if f != nil {
					slog.Info("became follower from leader state", "term", f.Term())
					st = fromFollower(f, r.timer)
				}
			default:
				return fmt.Errorf("unknown type in voteResponse section")
			}
		case <-r.timer.Chan():
			switch s := st.s.(type) {
			case *Follower:
				res := s.HandleElectionTimeout(r.handler)
				slog.Info("follower became a candidate after election timeout", "term", res.Term())
				st = fromCandidate(&res, r.timer)
			case *Candidate:
				res := s.HandleElectionTimeout(r.handler)
				slog.Info("candidate starts new round after election timeout", "term", res.Term())
				st = fromCandidate(&res, r.timer)
			case *Leader:
				s.SendHeartbeat(r.handler)
				slog.Debug("send heartbeat")
				st = fromLeader(s, r.timer)
			default:
				return fmt.Errorf("unknown type in timeout section")
			}
		}
	}
}
