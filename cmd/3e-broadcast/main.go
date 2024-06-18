package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type response struct {
	Type string `json:"type"`
}

type readResponse struct {
	Type     string `json:"type"`
	Messages []int  `json:"messages"`
}

type topologyRequest struct {
	Topology map[string][]string
}

type broadcastRequest struct {
	Type    string `json:"type"`
	Message int    `json:"message"`
}

func main() {
	var (
		messages     = make(map[int]struct{})
		messagesLock = sync.RWMutex{}

		neighbours = []string{}
	)

	n := maelstrom.NewNode()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var req broadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		messagesLock.RLock()
		_, exists := messages[req.Message]
		messagesLock.RUnlock()

		if !exists {
			for _, neighbour := range neighbours {
				if neighbour == msg.Src {
					continue
				}
				go func(neighbour string) {
					sent := false
					for !sent {
						n.RPC(neighbour,
							broadcastRequest{
								Type:    "broadcast",
								Message: req.Message,
							},
							func(msg maelstrom.Message) error {
								sent = true
								return nil
							})
						time.Sleep(time.Second)
					}
				}(neighbour)
			}
		}

		go func() {
			messagesLock.Lock()
			messages[req.Message] = struct{}{}
			messagesLock.Unlock()
		}()

		resp := response{
			Type: "broadcast_ok",
		}

		return n.Reply(msg, resp)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var msgs []int

		messagesLock.RLock()
		for msg := range messages {
			msgs = append(msgs, msg)
		}
		messagesLock.RUnlock()

		resp := readResponse{
			Type:     "read_ok",
			Messages: msgs,
		}

		return n.Reply(msg, resp)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var req topologyRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		neighbours = req.Topology[n.ID()]

		resp := response{
			Type: "topology_ok",
		}

		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
