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

type message struct {
	destination string
	number      int
}

func main() {
	var (
		messages     = make(map[int]struct{})
		messagesLock = sync.Mutex{}

		messagesToSend     = make(map[message]struct{})
		messagesToSendLock = sync.Mutex{}

		neighbours = []string{}
	)

	n := maelstrom.NewNode()

	// resend messages until success
	go func() {
		for {
			messagesToSendLock.Lock()
			for msgToSend := range messagesToSend {
				n.RPC(msgToSend.destination,
					broadcastRequest{
						Type:    "broadcast",
						Message: msgToSend.number,
					},
					func(msg maelstrom.Message) error {
						messagesToSendLock.Lock()
						delete(messagesToSend, msgToSend)
						messagesToSendLock.Unlock()

						return nil
					})
			}
			messagesToSendLock.Unlock()
			time.Sleep(time.Millisecond * 100)
		}
	}()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var req broadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		messagesLock.Lock()
		_, exists := messages[req.Message]
		messagesLock.Unlock()

		if !exists {
			for _, neighbour := range neighbours {
				if neighbour == msg.Src {
					continue
				}
				messagesToSendLock.Lock()
				messagesToSend[message{destination: neighbour, number: req.Message}] = struct{}{}
				messagesToSendLock.Unlock()

			}
		}

		messagesLock.Lock()
		messages[req.Message] = struct{}{}
		messagesLock.Unlock()

		resp := response{
			Type: "broadcast_ok",
		}

		return n.Reply(msg, resp)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var msgs []int

		messagesLock.Lock()
		for msg := range messages {
			msgs = append(msgs, msg)
		}
		messagesLock.Unlock()

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
