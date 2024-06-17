all: echo unique-ids broadcast-single broadcast-multi broadcast-fault-tolerant

build:
    nix build .#default

echo: build
    nix run .#maelstrom -- test -w echo --bin ./result/bin/echo --node-count 1 --time-limit 10

unique-ids: build
    nix run .#maelstrom -- test -w unique-ids --bin ./result/bin/unique-ids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

broadcast-single: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/broadcast --node-count 1 --time-limit 20 --rate 10

broadcast-multi: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/broadcast --node-count 5 --time-limit 20 --rate 10

broadcast-fault-tolerant: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/broadcast --node-count 5 --time-limit 20 --rate 10 --nemesis partition

broadcast-performance: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/broadcast --node-count 25 --time-limit 20 --rate 100 --latency 100 --topology tree4

    # Messages per operation
    cat store/latest/results.edn | jet -f '#(-> % :net :servers :msgs-per-op)'
    # Is it less than 30?
    cat store/latest/results.edn | jet -f '#(-> % :net :servers :msgs-per-op (< 30))'

    # Median latency
    cat store/latest/results.edn | jet -f '#(-> % :workload :stable-latencies (get 0.5))'
    # Is it less than 400ms
    cat store/latest/results.edn | jet -f '#(-> % :workload :stable-latencies (get 0.5) (< 400))'

    # Maximum latency
    cat store/latest/results.edn | jet -f '#(-> % :workload :stable-latencies (get 1))'
    # Is it less than 600ms?
    cat store/latest/results.edn | jet -f '#(-> % :workload :stable-latencies (get 1) (< 600))'
