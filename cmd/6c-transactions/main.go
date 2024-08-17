package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

// used to test G1a, but aborted transactions negatively affect the availability percentage that
// maelstrom calculates and causes the test to fail. So I've made this toggleable.
const abortTransactionsEnabled bool = false

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
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	var body txnRequest
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	txn := make([]operation, 0)

	// take snapshot of relevant keys before processing transaction
	snapshot := make(map[int]*int)
	for _, op := range body.Transaction {
		snapshot[op.key] = s.data[op.key]
	}

	for _, op := range body.Transaction {
		switch op.operationType {
		case read:
			op.value = s.data[op.key]
		case write:
			s.data[op.key] = op.value
		default:
			return errors.New(fmt.Sprintf("Unrecognized operation: %v", op.operationType))
		}

		txn = append(txn, op)
	}

	// randomly abort transactions from client messages
	if isClientMsg(msg) && abortTransactionsEnabled && shouldAbort() {
		// restore data from snapshot
		for k, v := range snapshot {
			s.data[k] = v
		}

		return s.n.Reply(msg,
			map[string]any{
				"type": "error",
				"code": maelstrom.TxnConflict,
				"text": "txn abort",
			})

	}

	// replicate to other nodes if transaction comes from a client
	if isClientMsg(msg) {
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

// returns true approximately 1% of the time
func shouldAbort() bool {
	n := rand.Intn(100)
	return n == 0
}

func isClientMsg(msg maelstrom.Message) bool {
	return strings.HasPrefix(msg.Src, "c")
}
