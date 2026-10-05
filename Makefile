.PHONY: graphs plots test

graphs:
	./graphing/generate.sh

plots:
	./graphing/generate.sh --plot-only

test:
	cd distributed-multi-workers && go test ./...
