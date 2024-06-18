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

type broadcastBatchRequest struct {
	Type    string `json:"type"`
	Message []int  `json:"message"`
}

func main() {
	var (
		messages     = make(map[int]struct{})
		messagesLock = sync.RWMutex{}

		messagesChan = make(chan int, 100)

		neighbours = []string{}
	)

	n := maelstrom.NewNode()

	// Background goroutine that fetches some messages from a channel and batch
	// sends them.
	go func() {
		for {
			msgBatch := make([]int, 0)
		L:
			for {
				select {
				case msg := <-messagesChan:
					msgBatch = append(msgBatch, msg)
				default:
					break L
				}
			}

			for _, neighbour := range neighbours {
				msg := broadcastBatchRequest{
					Type:    "broadcast_batch",
					Message: msgBatch,
				}
				go sendMessageWithRetry(n, neighbour, msg)
			}

			time.Sleep(time.Second)
		}
	}()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var req broadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		messagesLock.RLock()
		_, exists := messages[req.Message]
		messagesLock.RUnlock()

		if !exists {
			go func() {
				messagesChan <- req.Message
			}()
		}

		messagesLock.Lock()
		messages[req.Message] = struct{}{}
		messagesLock.Unlock()

		resp := response{
			Type: "broadcast_ok",
		}

		return n.Reply(msg, resp)
	})

	n.Handle("broadcast_batch", func(msg maelstrom.Message) error {
		var req broadcastBatchRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		messagesLock.RLock()
		allExists := true
		for _, message := range req.Message {
			_, exists := messages[message]
			if !exists {
				allExists = false
				break
			}

		}
		messagesLock.RUnlock()

		if !allExists {
			for _, neighbour := range neighbours {
				if neighbour == msg.Src {
					continue
				}
				message := broadcastBatchRequest{
					Type:    "broadcast_batch",
					Message: req.Message,
				}
				go sendMessageWithRetry(n, neighbour, message)
			}

		}

		messagesLock.Lock()
		for _, message := range req.Message {
			messages[message] = struct{}{}
		}
		messagesLock.Unlock()

		resp := response{
			Type: "broadcast_batch_ok",
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

func sendMessageWithRetry[T any](n *maelstrom.Node, dst string, message T) {
	sent := false
	for !sent {
		n.RPC(dst,
			message,
			func(msg maelstrom.Message) error {
				sent = true
				return nil
			})
		time.Sleep(time.Second)
	}

}
