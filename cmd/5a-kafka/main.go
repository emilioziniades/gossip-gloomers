package main

import (
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
	log         map[string][]int
	logMu       *sync.Mutex
	committed   map[string]int
	committedMu *sync.Mutex
}

func newServer() server {
	n := maelstrom.NewNode()
	return server{
		n:           n,
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
	offset := len(s.log[body.Key]) - 1

	s.logMu.Unlock()

	return s.n.Reply(msg, newSendResponse(offset))
}

func (s *server) poll(msg maelstrom.Message) error {
	var body pollRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.logMu.Lock()
	messages := poll(s.log, body.Offsets)
	s.logMu.Unlock()

	return s.n.Reply(msg, newPollResponse(messages))
}

func (s *server) commitOffsets(msg maelstrom.Message) error {
	var body commitOffsetsRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.committedMu.Lock()
	for key, offset := range body.Offsets {
		s.committed[key] = offset
	}
	s.committedMu.Unlock()

	return s.n.Reply(msg, newCommitOffsetsResponse())
}

func (s *server) listCommittedOffsets(msg maelstrom.Message) error {
	var body listCommittedOffsetsRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.committedMu.Lock()

	committed := make(map[string]int)
	for k, v := range s.committed {
		committed[k] = v
	}

	s.committedMu.Unlock()

	return s.n.Reply(msg, newListCommittedOffsetsResponse(committed))
}

func poll(data map[string][]int, offsets map[string]int) map[string][][]int {

	polled := make(map[string][][]int)

	for key, offset := range offsets {
		messages := make([]int, len(data[key])-offset)
		copy(messages, data[key][offset:])

		o := offset
		p := make([][]int, 0)

		for _, m := range messages {
			p = append(p, []int{o, m})
			o++

		}

		polled[key] = p
	}

	return polled
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

func newSendResponse(offset int) sendResponse {
	return sendResponse{
		Type:   "send_ok",
		Offset: offset,
	}
}

type pollRequest struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}

type pollResponse struct {
	Type     string             `json:"type"`
	Messages map[string][][]int `json:"msgs"`
}

func newPollResponse(messages map[string][][]int) pollResponse {
	return pollResponse{
		Type:     "poll_ok",
		Messages: messages,
	}
}

type commitOffsetsRequest struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}

type commitOffsetsResponse struct {
	Type string `json:"type"`
}

func newCommitOffsetsResponse() commitOffsetsResponse {
	return commitOffsetsResponse{
		Type: "commit_offsets_ok",
	}
}

type listCommittedOffsetsRequest struct {
	Type string   `json:"type"`
	Keys []string `json:"keys"`
}

type listCommittedOffsetsResponse struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}

func newListCommittedOffsetsResponse(offsets map[string]int) listCommittedOffsetsResponse {
	return listCommittedOffsetsResponse{
		Type:    "list_committed_offsets_ok",
		Offsets: offsets,
	}
}
