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

	s.n.Handle("add", s.add)
	s.n.Handle("read", s.read)

	if err := s.n.Run(); err != nil {
		log.Fatal(err)
	}

}

type server struct {
	n       *maelstrom.Node
	kv      *maelstrom.KV
	count   int
	countMu *sync.Mutex
}

func newServer() server {
	n := maelstrom.NewNode()
	return server{
		n:       n,
		kv:      maelstrom.NewSeqKV(n),
		count:   0,
		countMu: &sync.Mutex{},
	}
}

func (s *server) add(msg maelstrom.Message) error {
	var body addRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.countMu.Lock()
	err := s.kv.CompareAndSwap(context.TODO(), "counter", s.count, s.count+body.Delta, true)
	s.count += body.Delta
	s.countMu.Unlock()

	if err != nil {
		rpcErr, ok := err.(*maelstrom.RPCError)

		if !ok {
			return err
		}

		if rpcErr.Code != maelstrom.KeyDoesNotExist {
			log.Printf("KEY DOES NOT EXIST: %s", rpcErr.Text)
		} else if rpcErr.Code == maelstrom.PreconditionFailed {
			log.Printf("PRECONDITION FAILED: %s", rpcErr.Text)
		} else {
			log.Printf("OTHER: %s", rpcErr.Text)
			return err
		}

	}

	return s.n.Reply(msg, addResponse{Type: "add_ok"})
}

func (s *server) read(msg maelstrom.Message) error {
	n, err := s.kv.ReadInt(context.TODO(), "counter")

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
