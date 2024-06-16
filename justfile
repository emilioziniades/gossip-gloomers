test: test-echo

test-echo:
    nix build .#default
    nix run .#maelstrom -- test -w echo --bin ./result/bin/1-echo --node-count 1 --time-limit 10

