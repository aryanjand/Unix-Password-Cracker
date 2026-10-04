.PHONY: graphs test

graphs:
	./graphing/generate.sh

test:
	cd distributed-multi-workers && go test ./...
