package main

import (
	"github.com/Boar-D-White-Foundation/kvgo"
	"log/slog"
	"time"
)

type appendEntryMessage struct {
	request  kvgo.AppendEntryRequest
	response chan kvgo.AppendEntryResponse
}

type voteMessage struct {
	request  kvgo.VoteRequest
	response chan kvgo.VoteResponse
}

type Raft struct {
	cfg                  kvgo.Config
	appendEntries        chan appendEntryMessage
	votes                chan voteMessage
	appendEntryResponses chan kvgo.AppendEntryResponse
	voteResponses        chan kvgo.VoteResponse
	terminate            chan bool
}

func MakeRaftService(cfg kvgo.Config) Raft {
	return Raft{
		cfg:                  cfg,
		appendEntries:        make(chan appendEntryMessage),
		votes:                make(chan voteMessage),
		appendEntryResponses: make(chan kvgo.AppendEntryResponse),
		voteResponses:        make(chan kvgo.VoteResponse),
	}
}

type state struct {
	s interface{}
	t <-chan time.Time
}

func beginState(cfg *kvgo.Config) state {
	f := kvgo.MakeFollower(cfg, 0)
	return state{
		s: &f,
		t: time.After(f.Timeout()),
	}
}

func fromCandidate(c *kvgo.Candidate) state {
	return state{
		s: c,
		t: time.After(c.Timeout()),
	}
}

func fromFollower(f *kvgo.Follower) state {
	return state{
		s: f,
		t: time.After(f.Timeout()),
	}
}

func fromLeader(l *kvgo.Leader) state {
	return state{
		s: l,
		t: time.After(l.Timeout()),
	}
}

func (r *Raft) Stop() {
	r.terminate <- true
}

func (r *Raft) Start() {
	st := beginState(&r.cfg)
	for {
		select {
		case <-r.terminate:
			slog.Info("main loop stopped")
			return
		case appendEntry := <-r.appendEntries:
			switch s := st.s.(type) {
			case *kvgo.Follower:
				f, resp := s.HandleAppendEntry(appendEntry.request)
				appendEntry.response <- resp
				if f != nil {
					slog.Info("follow for new leader", "term", f.Term())
					st = fromFollower(f)
				} else {
					// reset timeout
					st = fromFollower(s)
				}
			case *kvgo.Candidate:
				f, resp := s.HandleAppendEntry(appendEntry.request)
				appendEntry.response <- resp
				if f != nil {
					slog.Info("became follower from candidate state", "term", f.Term())
					st = fromFollower(f)
				}
			case *kvgo.Leader:
				f, resp := s.HandleAppendEntry(appendEntry.request)
				appendEntry.response <- resp
				if f != nil {
					slog.Info("became follower from leader state", "term", f.Term())
					st = fromFollower(f)
				}
			default:
				slog.Error("unknown type in appendEntries section")
				return
			}
		case vote := <-r.votes:
			switch s := st.s.(type) {
			case *kvgo.Follower:
				f, resp := s.HandleVote(vote.request)
				vote.response <- resp
				if f != nil {
					slog.Info("follow for new leader", "term", f.Term())
					if resp.VoteGranted {
						slog.Info("granted vote for", "term", f.VoteFor())
					}
					st = fromFollower(f)
				}
			case *kvgo.Candidate:
				f, resp := s.HandleVote(vote.request)
				vote.response <- resp
				if f != nil {
					slog.Info("became follower from candidate state", "term", f.Term())
					if resp.VoteGranted {
						slog.Info("granted vote for", "term", f.VoteFor())
					}
					st = fromFollower(f)
				}
			case *kvgo.Leader:
				f, resp := s.HandleVote(vote.request)
				vote.response <- resp
				if f != nil {
					slog.Info("became follower from leader state", "term", f.Term())
					if resp.VoteGranted {
						slog.Info("granted vote for", "term", f.VoteFor())
					}
					st = fromFollower(f)
				}
			default:
				slog.Error("unknown type in votes section")
				return
			}
		case voteResp := <-r.voteResponses:
			switch s := st.s.(type) {
			case *kvgo.Follower:
				slog.Debug("skip vote response for follower")
			case *kvgo.Candidate:
				l, f := s.HandleVoteResponse(voteResp, r)
				if f != nil {
					slog.Info("became follower from candidate state", "term", f.Term())
					st = fromFollower(f)
				}
				if l != nil {
					slog.Info("became a leader", l.Term())
					st = fromLeader(l)
				}
			case *kvgo.Leader:
				slog.Debug("skip vote response for leader")
			default:
				slog.Error("unknown type in voteResponse section")
				return
			}
		case _ = <-r.appendEntryResponses:
			// skip
		case <-st.t:
			switch s := st.s.(type) {
			case *kvgo.Follower:
				res := s.HandleElectionTimeout(r)
				slog.Info("follower became a candidate after election timeout", "term", res.Term())
				st = fromCandidate(&res)
			case *kvgo.Candidate:
				res := s.HandleElectionTimeout(r)
				slog.Info("candidate starts new round after election timeout", "term", res.Term())
				st = fromCandidate(&res)
			case *kvgo.Leader:
				s.SendHeartbeat(r)
				slog.Debug("send heartbeat")
				st = fromLeader(s)
			default:
				slog.Error("unknown type in timeout section")
				return
			}
		}
	}
}
