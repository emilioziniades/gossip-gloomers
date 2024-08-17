package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	primaryId  string = "n0"
	counterKey string = "counter"
)

func main() {
	s := newServer()

	s.n.Handle("add", s.add)
	s.n.Handle("read", s.read)

	if err := s.n.Run(); err != nil {
		log.Fatal(err)
	}
}

type server struct {
	n   *maelstrom.Node
	kv  *maelstrom.KV
	c   int
	cMu *sync.Mutex
}

func newServer() server {
	n := maelstrom.NewNode()
	return server{
		n:   n,
		kv:  maelstrom.NewSeqKV(n),
		c:   0,
		cMu: &sync.Mutex{},
	}
}

func (s *server) add(msg maelstrom.Message) error {
	var body addRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.cMu.Lock()
	err := s.kv.CompareAndSwap(context.Background(), counterKey, s.c, s.c+body.Delta, true)
	s.c += body.Delta
	s.cMu.Unlock()

	// broadcast add to other nodes if message comes from client
	if strings.HasPrefix(msg.Src, "c") {
		for _, nId := range s.n.NodeIDs() {
			if nId == s.n.ID() {
				continue
			}

			// retry in a separate goroutine
			go func(nId string) {
				sent := false
				for !sent {
					s.n.RPC(nId, body, func(msg maelstrom.Message) error {
						sent = true
						return nil
					})
					time.Sleep(2 * time.Second)
				}
			}(nId)

		}
	}

	if err != nil {
		rpcErr, ok := err.(*maelstrom.RPCError)

		if !ok {
			return err
		}

		if rpcErr.Code != maelstrom.KeyDoesNotExist && rpcErr.Code != maelstrom.PreconditionFailed {
			return err
		}

	}

	return s.n.Reply(msg, addResponse{Type: "add_ok"})
}

func (s *server) read(msg maelstrom.Message) error {
	n, err := s.kv.ReadInt(context.Background(), counterKey)
	if err != nil {
		return err
	}

	return s.n.Reply(msg, readResponse{Type: "read_ok", Value: n})
}

type readRequest struct {
	Type string `json:"type"`
}

type readResponse struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type addRequest struct {
	Type  string `json:"type"`
	Delta int    `json:"delta"`
}

type addResponse struct {
	Type string `json:"type"`
}
