package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	s := newServer()

	s.n.Handle("send", s.send)
	s.n.Handle("poll", s.poll)
	s.n.Handle("commit_offsets", s.commitOffsets)
	s.n.Handle("list_committed_offsets", s.listCommittedOffsets)

	if err := s.n.Run(); err != nil {
		log.Fatal(err)
	}
}

type server struct {
	n           *maelstrom.Node
	linKV       *maelstrom.KV
	seqKV       *maelstrom.KV
	log         map[string][]int
	logMu       *sync.RWMutex
	committed   map[string]int
	committedMu *sync.RWMutex
}

func newServer() server {
	n := maelstrom.NewNode()
	linKV := maelstrom.NewLinKV(n)
	seqKV := maelstrom.NewSeqKV(n)
	return server{
		n:           n,
		linKV:       linKV,
		seqKV:       seqKV,
		log:         make(map[string][]int),
		logMu:       &sync.RWMutex{},
		committed:   make(map[string]int),
		committedMu: &sync.RWMutex{},
	}
}

func (s *server) send(msg maelstrom.Message) error {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	var body sendRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	if s.isPrimary() {
		// update local state and write to kv store
		s.log[body.Key] = append(s.log[body.Key], body.Message)

		logs := s.log[body.Key]
		oldLogs := logs[0 : len(logs)-1]
		offset := len(logs) - 1

		if err := s.linKV.CompareAndSwap(context.Background(), body.Key, oldLogs, logs, true); err != nil {
			log.Println("ERROR send", body.Key, err)
		}

		return s.n.Reply(msg, sendResponse{Type: "send_ok", Offset: offset})
	} else {
		// ask primary to update kv store and return response
		m, err := s.n.SyncRPC(context.TODO(), s.getPrimary(), body)
		if err != nil {
			log.Println("ERROR send-primary", err)
		}

		return s.n.Reply(msg, m.Body)
	}
}

func (s *server) poll(msg maelstrom.Message) error {
	s.logMu.RLock()
	defer s.logMu.RUnlock()

	var body pollRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	polled := make(map[string][][]int)

	for key, offset := range body.Offsets {
		messages := make([]int, 0)
		if err := s.linKV.ReadInto(context.Background(), key, &messages); err != nil {
			log.Println("ERROR poll", key, err)
		}

		messages = messages[offset:]

		o := offset
		p := make([][]int, 0)

		for _, m := range messages {
			p = append(p, []int{o, m})
			o++

		}

		polled[key] = p
	}

	return s.n.Reply(msg, pollResponse{Type: "poll_ok", Messages: polled})
}

func (s *server) commitOffsets(msg maelstrom.Message) error {
	s.committedMu.Lock()
	defer s.committedMu.Unlock()

	var body commitOffsetsRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	if s.isPrimary() {
		// update local state and write to kv store
		for key, offset := range body.Offsets {
			oldOffset := s.committed[key]
			s.committed[key] = offset
			if err := s.seqKV.CompareAndSwap(context.Background(), "committed-"+key, oldOffset, offset, true); err != nil {
				log.Println("ERROR commitOffsets", "committed-"+key, err)
			}
		}
	} else {
		// ask primary to update kv store and return response
		s.n.SyncRPC(context.TODO(), s.getPrimary(), body)
	}

	return s.n.Reply(msg, commitOffsetsResponse{Type: "commit_offsets_ok"})
}

func (s *server) listCommittedOffsets(msg maelstrom.Message) error {
	s.committedMu.RLock()
	defer s.committedMu.RUnlock()

	var body listCommittedOffsetsRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	committed := make(map[string]int)
	for _, key := range body.Keys {
		offset, err := s.seqKV.ReadInt(context.Background(), "committed-"+key)
		if err != nil && maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
			log.Println("ERROR listCommittedOffsets", "committed-"+key, err)
		}
		committed[key] = offset
	}

	return s.n.Reply(msg, listCommittedOffsetsResponse{Type: "list_committed_offsets_ok", Offsets: committed})
}

// A simple way to establish a primary node. `n0` is always the primary.
func (s *server) isPrimary() bool {
	return s.n.ID() == "n0"
}

func (s *server) getPrimary() string {
	return "n0"
}

type sendRequest struct {
	Type    string `json:"type"`
	Key     string `json:"key"`
	Message int    `json:"msg"`
}

type sendResponse struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
}

type pollRequest struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}

type pollResponse struct {
	Type     string             `json:"type"`
	Messages map[string][][]int `json:"msgs"`
}

type commitOffsetsRequest struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}

type commitOffsetsResponse struct {
	Type string `json:"type"`
}

type listCommittedOffsetsRequest struct {
	Type string   `json:"type"`
	Keys []string `json:"keys"`
}

type listCommittedOffsetsResponse struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}
