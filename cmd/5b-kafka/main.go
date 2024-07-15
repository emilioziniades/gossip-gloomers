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
	kv          *maelstrom.KV
	log         map[string][]int
	logMu       *sync.Mutex
	committed   map[string]int
	committedMu *sync.Mutex
}

func newServer() server {
	n := maelstrom.NewNode()
	kv := maelstrom.NewLinKV(n)
	return server{
		n:           n,
		kv:          kv,
		log:         make(map[string][]int),
		logMu:       &sync.Mutex{},
		committed:   make(map[string]int),
		committedMu: &sync.Mutex{},
	}
}

func (s *server) send(msg maelstrom.Message) error {
	var body sendRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.logMu.Lock()

	s.log[body.Key] = append(s.log[body.Key], body.Message)

	logs := s.log[body.Key]
	oldLogs := make([]int, len(logs)-1)
	newLogs := make([]int, len(logs))
	copy(oldLogs, logs)
	copy(newLogs, logs)

	offset := len(logs) - 1

	if err := s.kv.CompareAndSwap(context.Background(), body.Key, oldLogs, newLogs, true); err != nil {
		log.Println("ERROR send", body.Key, err)
	}

	s.logMu.Unlock()

	// broadcast to other nodes if "send" comes from a client
	// if strings.HasPrefix(msg.Src, "c") {
	// 	for _, n := range s.n.NodeIDs() {
	// 		if n == s.n.ID() {
	// 			continue
	// 		}
	// 		if err := s.n.RPC(n, body, func(msg maelstrom.Message) error { return nil }); err != nil {
	// 			// return err
	// 		}
	// 	}
	// }

	return s.n.Reply(msg, sendResponse{Type: "send_ok", Offset: offset})
}

func (s *server) poll(msg maelstrom.Message) error {
	var body pollRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.logMu.Lock()

	polled := make(map[string][][]int)

	for key, offset := range body.Offsets {
		messages := make([]int, 0)
		if err := s.kv.ReadInto(context.Background(), key, &messages); err != nil {
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

	s.logMu.Unlock()

	return s.n.Reply(msg, pollResponse{Type: "poll_ok", Messages: polled})
}

func (s *server) commitOffsets(msg maelstrom.Message) error {
	var body commitOffsetsRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.committedMu.Lock()

	for key, offset := range body.Offsets {
		oldOffset := s.committed[key]
		s.committed[key] = offset
		if err := s.kv.CompareAndSwap(context.Background(), "committed-"+key, oldOffset, offset, true); err != nil {
			log.Println("ERROR commitOffsets", "committed-"+key, err)
		}

	}

	s.committedMu.Unlock()

	return s.n.Reply(msg, commitOffsetsResponse{Type: "commit_offsets_ok"})
}

func (s *server) listCommittedOffsets(msg maelstrom.Message) error {
	var body listCommittedOffsetsRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.committedMu.Lock()

	committed := make(map[string]int)
	for _, key := range body.Keys {
		offset, err := s.kv.ReadInt(context.Background(), "committed-"+key)
		if err != nil && maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
			log.Println("ERROR listCommittedOffsets", "committed-"+key, err)
		}
		committed[key] = offset
	}

	s.committedMu.Unlock()

	return s.n.Reply(msg, listCommittedOffsetsResponse{Type: "list_committed_offsets_ok", Offsets: committed})
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
