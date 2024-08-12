# Gossip Gloomers Puzzles

My solutions to [Gossip Gloomers](https://fly.io/dist-sys/)- the Fly.io distributed systems challenges.

TODO: summary of challenge (uses maelstrom etc)
TODO: Note about setup using nix
TODO: link to article on my blog

# 2: Unique ID Generation

TODO: link to solution

To generate globally unique IDs, I concatenate the node ID and a monotonically increasing counter together.

# 3: Broadcast

## 3a: Single-Node Broadcast

TODO: link to solution

## 3b: Multi-Node Broadcast

TODO: link to solution

Code is the same as for 3a.
Nothing particularly interesting.
The node broadcasts the message it receives to all its neighbours.
I ended up using the default grid topology provided by maelstrom.

## 3c: Fault Tolerant Broadcast

TODO: link to solution

Now that network calls can fail, I built a retry mechanism into the node.
In a separate goroutine, a message is resent every second until the destination node acknowledges the broadcast.

Initially I had a global list of messages to retry, but this led to lock contention.
After reworking it, the handler just fires off a goroutine, and that goroutine is responsible for retrying the message until it is acknowledged.

## 3d: Efficient Broadcast, Part 1

TODO: link to solution

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

In future, I could build my own spanning tree out of a fully connected graph of nodes.

## 3e: Efficient Broadcast, Part 2

TODO: link to solution

This challenge relaxes the requirement on latency, and makes the messages per operation stricter.

\<digression>
After doing this challenge, it is clear that there is a tradeoff between latency and message rate.

In a fully connected (total) topology, latency is very low because each node is connected to every other node.
However, a lot of duplicate messages will be sent.

On the other hand, in a line topology, where each node is connected to only two other nodes, there will be no duplicate messages.
This is also true in minimum spanning trees.
But, the latency is high in a line topology because the distance between nodes may be large.
\</digression>

In any case, the challenge asks for a maximum of 20 messages per operation.
This is impossible if you broadcast one operation at a time, because it takes at least 24 messages to broadcast one operation to 25 nodes.

So, I opted for batching messages together. A goroutine runs in a loop once a second, fetching pending messages from a channel, and broadcasts in batches. Sticking with the spanning tree topology, the system performed as follows:

- Messages per operation: 12.77
- Median stable latency: 853ms
- Maximum stable latency: 1411ms

## 4: Grow-Only Counter

TODO: link to solution

The key to my solution was that each node kept a local counter, and only did compare-and-swaps against the global counter in the seq-kv store.

When a node receives an add message, it broadcasts that add to all other nodes.
To prevent a broadcast storm, adds are only broacast if the message comes from a client, and not another node.
To avoid race conditions, I only mutate the counter in the key-value store with compare-and-swaps.
The node keeps a local counter for compare-and-swaps.
All nodes try perform the same compare-and-swaps. This is safe because writes will only occur if the node has an accurate view of the state of the global counter.
These broadcasts are also retried until a response is received, in case messages fail during network partitions.

This is enough to pass the challenge.
Honestly, I'm still not fully clear on why this works.
I think my lack of understanding of sequential consistency is preventing me from intuitively understanding the solution.

## 5a: Single-Node Kafka Logs

TODO: link to solution

Since it's only one node, the challenge is pretty straightforward.
Putting mutexes around all the core data structures is muscle memory at this point.

I was quite pleased with the data structures I picked, and a lot of the code flowed after I got these correct.
`server.log` was a `map[string][]int`.
Each key was an entry in the map, and the arrays of integers were the messages.
Offsets were defined as the message index in the array.
This made offset-based lookups fast (O(1)) because it is just a slice index.
Log appends were also fast.
Those were just slice appends (also O(1)).

## 5b: Multi-Node Kafka Logs

TODO: link to solution

This challenge is trickier than the previous one.

After some experimenting, I've concluded that the key-value stores provided by maelstrom are global.
If one node writes to a key, all other nodes can read from that key.
This seems obvious in hindsight, but it wasn't so clear in challenge 4.

The biggest difference between challenges 4 and 5 is the data structure we are persisting to the key-value store.

In challenge 4, the data being persisted is a grow-only counter.
The adds could occur in any order as long as they all occured, and this allowed a simple broadcast of all adds, and letting each node safely CAS the result.

In challenge 5, the data being persisted is multiple arrays.
This challenge is not so simple, because the array appends _must_ occur in the same order.
This means broadcasting and free-for-all CASs does not work.

Instead, I opted for a primary-secondary setup, where `n0` is always the primary, and all other nodes are secondaries.
Only the primary writes to the key-value store.
If a secondary receives a write operation (`send` or `commit_offsets`), it passes the write onto the primary and returns the response.
For reads, secondaries read directly from the key-value store.

Based on about 5 minutes reading time, it seems like the real Kafka has a similar architecture, with leaders and replicas.
There are a lot (_a lot_) of complexities I don't have to deal with, such as replication, leader elections and sharding.
All nodes (primary or secondary) read from the lin-kv store, so the primary does not have to replicate its data out.
And because this system is really simple, the primary remains `n0` and I just don't bother with an election.

Admittedly, network partitions would be troublesome, as all writes would fail if a node couldn't reach the primary.
No progress could be made on writes until the network partitions are resolved.
I am opting for consistency over availability (for writes) if there is a partition.
Luckily, this challenge does not include network partitions.
