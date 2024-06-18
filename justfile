all: echo unique-ids broadcast-single broadcast-multi broadcast-fault-tolerant

build:
    nix build .#default

# 1
echo: build
    nix run .#maelstrom -- test -w echo --bin ./result/bin/1-echo --node-count 1 --time-limit 10

# 2
unique-ids: build
    nix run .#maelstrom -- test -w unique-ids --bin ./result/bin/2-unique-ids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

# 3a
broadcast-single: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/3a-broadcast --node-count 1 --time-limit 20 --rate 10

# 3b
broadcast-multi: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/3b-broadcast --node-count 5 --time-limit 20 --rate 10

# 3c
broadcast-fault-tolerant: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/3c-broadcast --node-count 5 --time-limit 20 --rate 10 --nemesis partition

# 3d
broadcast-performance: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/3d-broadcast --node-count 25 --time-limit 20 --rate 100 --latency 100 --topology tree4
    cat store/latest/results.edn | jet -f queries/3d-broadcast.clj
