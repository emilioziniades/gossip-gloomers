# Gossip Gloomers Puzzles

My solutions to [Gossip Gloomers](https://fly.io/dist-sys/)- the Fly.io distributed systems challenges

TODO: summary of challenge (uses maelstrom etc)
TODO: Note about setup using nix
TODO: link to article on my blog

# 2: Unique ID Generation

TODO: link to solution

TODO: put mutex around counter

To generate globally unique IDs, I concatenate the node ID and a monotonically increasing counter together.

# 3: Broadcast

## 3a: Single-Node Broadcast

TODO: link to solution

## 3b: Multi-Node Broadcast

TODO: link to solution
TODO: split out code into separate parts

Code is the same as for 3a.
Nothing particularly interesting.
The node broadcasts the message it receives to all its neighbours.
I ended up using the default grid topology provided by maelstrom.

## 3c: Fault Tolerant Broadcast

TODO: link to solution
TODO: split out code into separate parts

Now that network calls can fail, I built a retry mechanism into the node.
In a separate goroutine, a message is resent every second until the destination node acknowledges the broadcast.

Initially I had a global list of messages to retry, but this led to lock contention.
After reworking it, the handler just fires off a goroutine, and that goroutine is responsible for retrying the message until it is acknowledged.

## 3d: Efficient Broadcast, Part 1

TODO: link to solution
TODO: split out code into separate parts

The code is exactly the same as 3c.
The only difference is the network topology.

Instead of building a topology in each node's initialization, I just made use of maelstrom's `--topology` flag.

I opted for a spanning tree (`--topology tree4`).
This topology is the most optimal for both broadcast latency and messages per operation.
It takes 24 messages to broadcast to 25 nodes.
There are no duplicate messages sent to the same node.
In addition, in `tree4`, each node has at most 4 edges, so with 25 nodes and 100ms latency, it should never take longer than 600ms to broadcast a message to all nodes.

Just changing the topology was enough to satisfy the challenges performance requirements. It performed as follows:

- Messages per operation: 23.72 (slightly below 24 because the read operations don't require broadcasting)
- Median stable latency: 377ms
- Maximum stable latency: 522ms

## 3e: Efficient Broadcast, Part 2

TODO: link to solution
TODO: split out code into separate parts

This challenge relaxes the requirement on latency, and makes the messages per operation stricter.

After doing this challenge, it is clear that there is a tradeoff between latency and messages per operation.

In a fully connected (total) topology, latency is very low because each node is connected to every other node.
However, a lot of duplicate messages will be sent.

On the other hand, in a line topology, where each node is connected to only two other nodes, there will be no duplicate messages.
This is also true in minimum spanning trees.
But, the latency is high in a line topology because the distance between nodes may be large.

In any case, the challenge asks for a maximum of 20 messages per operation.
This is impossible if you broadcast one message at a time, because it takes at least 24 messages to broadcast to 25 nodes.

So, I opted for batching messages together. A goroutine runs in a loop, fetching 5 messages from a channel, and broadcasts in batches of 5. Sticking with the spanning tree topology, the system performed as follows:

TODO stick in the system performance (once you have actually written the code haha)
