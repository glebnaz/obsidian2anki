BINARY  = obs2anki
OUT_DIR = .bin

.PHONY: build clean test

build:
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/$(BINARY) ./cmd/obs2anki

test:
	go test ./...

clean:
	rm -rf $(OUT_DIR)
