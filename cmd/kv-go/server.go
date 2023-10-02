package main

import (
	"fmt"
	kv_go "github.com/Boar-D-White-Foundation/kv-go"
	"log"
	"time"
)

type appendEntryMessage struct {
	request  kv_go.AppendEntryRequest
	response chan kv_go.AppendEntryResponse
}

type voteMessage struct {
	request  kv_go.VoteRequest
	response chan kv_go.VoteResponse
}

type Raft struct {
	cfg                  kv_go.Config
	appendEntries        chan appendEntryMessage
	votes                chan voteMessage
	appendEntryResponses chan kv_go.AppendEntryResponse
	voteResponses        chan kv_go.VoteResponse
	terminate            chan bool
}

func MakeRaftService(cfg kv_go.Config) Raft {
	return Raft{
		cfg:                  cfg,
		appendEntries:        make(chan appendEntryMessage),
		votes:                make(chan voteMessage),
		appendEntryResponses: make(chan kv_go.AppendEntryResponse),
		voteResponses:        make(chan kv_go.VoteResponse),
	}
}

type state struct {
	s interface{}
	t <-chan time.Time
}

func beginState(cfg *kv_go.Config) state {
	f := kv_go.MakeFollower(cfg, 0)
	return state{
		s: &f,
		t: time.After(f.Timeout()),
	}
}

func fromCandidate(c *kv_go.Candidate) state {
	return state{
		s: c,
		t: time.After(c.Timeout()),
	}
}

func fromFollower(f *kv_go.Follower) state {
	return state{
		s: f,
		t: time.After(f.Timeout()),
	}
}

func fromLeader(l *kv_go.Leader) state {
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
			log.Println("[INFO] main loop stopped")
			return
		case appendEntry := <-r.appendEntries:
			switch s := st.s.(type) {
			case *kv_go.Follower:
				resp := s.HandleAppendEntry(appendEntry.request)
				if !resp.Success {
					log.Println("[DEBUG] can't accept AppendEntry RPC")
				} else {
					st = fromFollower(s)
				}
				appendEntry.response <- resp
			case *kv_go.Candidate:
				res, resp := s.HandleAppendEntry(appendEntry.request)
				if !resp.Success {
					log.Println("[DEBUG] can't accept AppendEntry RPC")
				} else {
					st = fromFollower(res)
					log.Println("[INFO] became follower from candidate state")
				}
				appendEntry.response <- resp
			case *kv_go.Leader:
				res, resp := s.HandleAppendEntry(appendEntry.request)
				if !resp.Success {
					log.Println("[DEBUG] can't accept AppendEntry RPC")
				} else {
					st = fromFollower(res)
					log.Println("[INFO] became follower from leader state")
				}
				appendEntry.response <- resp
			default:
				log.Println("[ERROR] unknown type in appendEntries section")
				return
			}
		case vote := <-r.votes:
			switch s := st.s.(type) {
			case *kv_go.Follower:
				resp := s.HandleVote(vote.request)
				if resp.VoteGranted {
					log.Println(fmt.Sprintf("[INFO] granted vote for %v", vote.request.CandidateId))
				} else {
					log.Println(fmt.Sprintf("[TRACE] vote is not granted for %v", vote.request.CandidateId))
				}
				vote.response <- resp
			case *kv_go.Candidate:
				res, resp := s.HandleVote(vote.request)
				if resp.VoteGranted {
					log.Println(fmt.Sprintf("[INFO] granted vote for %v and became follower", vote.request.CandidateId))
					st = fromFollower(res)
				} else {
					log.Println(fmt.Sprintf("[TRACE] skip vote requests for other candidates"))
				}
				vote.response <- resp
			case *kv_go.Leader:
				resp := s.HandleVote(vote.request)
				if resp.VoteGranted {
					log.Println(fmt.Sprintf("[INFO] granted vote for %v", vote.request.CandidateId))
				} else {
					log.Println(fmt.Sprintf("[TRACE] vote is not granted for %v", vote.request.CandidateId))
				}
				vote.response <- resp
			default:
				log.Println("[ERROR] unknown type in votes section")
				return
			}
		case voteResp := <-r.voteResponses:
			switch s := st.s.(type) {
			case *kv_go.Follower:
				log.Println("[TRACE] skip vote response for follower")
			case *kv_go.Candidate:
				res, err := s.HandleVoteResponse(voteResp, r)
				if err != nil {
					log.Println(fmt.Sprintf("[TRACE] can't became a leader yet: %s", err))
					continue
				}
				log.Println("[INFO] became a leader")
				st = fromLeader(res)
			case *kv_go.Leader:
				log.Println("[TRACE] skip vote response for leader")
			default:
				log.Println("[ERROR] unknown type in voteResponse section")
				return
			}
		case <-st.t:
			log.Println("[INFO] timeout")
			switch s := st.s.(type) {
			case *kv_go.Follower:
				res := s.HandleElectionTimeout(r)
				log.Println("[INFO] follower became a candidate after election timeout")
				st = fromCandidate(&res)
			case *kv_go.Candidate:
				res := s.HandleElectionTimeout(r)
				log.Println("[INFO] candidate starts new round after election timeout")
				st = fromCandidate(&res)
			case *kv_go.Leader:
				s.SendHeartbeat(r)
				log.Println("[TRACE] send heartbeat")
				st = fromLeader(s)
			default:
				log.Println("[ERROR] unknown type in timeout section")
				return
			}
		}
	}
}
