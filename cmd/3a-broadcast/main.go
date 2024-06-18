package main

import (
	"encoding/json"
	"log"
	"sync"

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
		messages     = make([]int, 0)
		messagesLock = sync.Mutex{}
	)

	n := maelstrom.NewNode()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var req broadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		messagesLock.Lock()
		messages = append(messages, req.Message)
		messagesLock.Unlock()

		resp := response{
			Type: "broadcast_ok",
		}

		return n.Reply(msg, resp)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		resp := readResponse{
			Type:     "read_ok",
			Messages: messages,
		}

		return n.Reply(msg, resp)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var req topologyRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		resp := response{
			Type: "topology_ok",
		}

		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
