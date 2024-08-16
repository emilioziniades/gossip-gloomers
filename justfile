all: echo unique-ids broadcast-single broadcast-multi broadcast-fault-tolerant broadcast-performance counter kafka-single kafka-multi

maelstrom workflow binary arguments:
    nix develop -c bash -c 'BIN=$(command -v {{ binary }}); maelstrom test -w {{ workflow }} --bin $BIN {{ arguments }}'

# 1
echo:
    just maelstrom echo 1-echo '--node-count 1 --time-limit 10'

# 2
unique-ids:
    just maelstrom unique-ids 2-unique-ids '--time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition'

# 3a
broadcast-single:
    just maelstrom broadcast 3a-broadcast '--node-count 1 --time-limit 20 --rate 10'

# 3b
broadcast-multi:
    just maelstrom broadcast 3b-broadcast '--node-count 5 --time-limit 20 --rate 10'

# 3c
broadcast-fault-tolerant:
    just maelstrom broadcast 3c-broadcast '--node-count 5 --time-limit 20 --rate 10 --nemesis partition'

# 3d
broadcast-performance:
    just maelstrom broadcast 3d-broadcast '--node-count 25 --time-limit 20 --rate 100 --latency 100 --topology tree4'
    cat store/latest/results.edn | jet -f queries/3d-broadcast.clj

# 3e
broadcast-performance-again:
    just maelstrom broadcast 3e-broadcast '--node-count 25 --time-limit 20 --rate 100 --latency 100 --topology tree4'
    cat store/latest/results.edn | jet -f queries/3e-broadcast.clj

# 4
counter:
    just maelstrom g-counter 4-counter '--node-count 3 --rate 100 --time-limit 20 --nemesis partition'

# 5a
kafka-single:
    just maelstrom kafka 5a-kafka '--node-count 1 --concurrency 2n --time-limit 20 --rate 1000'

# 5b
kafka-multi:
    just maelstrom kafka 5b-kafka '--node-count 2 --concurrency 2n --time-limit 20 --rate 1000'

# 5c
kafka-multi-optimized:
    just maelstrom kafka 5c-kafka '--node-count 2 --concurrency 2n --time-limit 20 --rate 1000'

# 6a
transactions-single:
    just maelstrom txn-rw-register 6a-transactions '--node-count 1 --time-limit 20 --rate 1000 --concurrency 2n --consistency-models read-uncommitted --availability total'

# 6b
transactions-multi:
    just maelstrom txn-rw-register 6b-transactions '--node-count 2 --time-limit 20 --rate 1000 --concurrency 2n --consistency-models read-uncommitted --availability total --nemesis partition'

# 6c
transactions-multi-read-committed:
    just maelstrom txn-rw-register 6c-transactions '--node-count 2 --time-limit 20 --rate 1000 --concurrency 2n --consistency-models read-committed --availability total --nemesis partition'
