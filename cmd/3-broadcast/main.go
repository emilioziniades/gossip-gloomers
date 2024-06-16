package main

import (
	"encoding/json"
	"fmt"
	"log"

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
		messages = make(map[int]struct{})
		topology = make(map[string][]string)
	)

	n := maelstrom.NewNode()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var req broadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		if _, exists := messages[req.Message]; !exists {
			for neighbour := range topology[n.ID()] {
				neighbourId := fmt.Sprintf("n%d", neighbour)
				n.Send(neighbourId, req)
			}
		}

		messages[req.Message] = struct{}{}

		resp := response{
			Type: "broadcast_ok",
		}
		return n.Reply(msg, resp)
	})

	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var msgs []int
		for msg := range messages {
			msgs = append(msgs, msg)
		}

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

		topology = req.Topology
		log.Println(topology)

		resp := response{
			Type: "topology_ok",
		}

		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
