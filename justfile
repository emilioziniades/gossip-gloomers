test: test-echo test-unique-ids

build:
    nix build .#default

test-echo: build
    nix run .#maelstrom -- test -w echo --bin ./result/bin/1-echo --node-count 1 --time-limit 10

test-unique-ids: build
    nix run .#maelstrom -- test -w unique-ids --bin ./result/bin/2-unique-ids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

test-broadcast-single: build
    nix run .#maelstrom -- test -w broadcast --bin ./result/bin/3-broadcast --node-count 1 --time-limit 20 --rate 10
