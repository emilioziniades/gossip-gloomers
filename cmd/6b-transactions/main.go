package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	s := newServer()

	s.n.Handle("txn", s.txn)

	if err := s.n.Run(); err != nil {
		log.Fatal(err)
	}
}

type server struct {
	n      *maelstrom.Node
	data   map[int]*int
	dataMu *sync.Mutex
}

func newServer() server {
	n := maelstrom.NewNode()
	return server{
		n:      n,
		data:   make(map[int]*int),
		dataMu: &sync.Mutex{},
	}
}

func (s *server) txn(msg maelstrom.Message) error {
	var body txnRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	txn := make([]operation, 0)

	s.dataMu.Lock()

	for _, op := range body.Transaction {
		switch op.operationType {
		case read:
			v := s.data[op.key]
			op.value = v
		case write:
			s.data[op.key] = op.value
		default:
			return errors.New(fmt.Sprintf("Unrecognized operation: %v", op.operationType))
		}

		txn = append(txn, op)
	}

	s.dataMu.Unlock()

	// replicate to other nodes if transaction comes from a client
	if strings.HasPrefix(msg.Src, "c") {
		for _, nId := range s.n.NodeIDs() {
			// do not send to self
			if s.n.ID() == nId {
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

	return s.n.Reply(msg, txnResponse{Type: "txn_ok", Transaction: txn})
}
