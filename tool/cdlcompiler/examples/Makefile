.PHONY: build generate clean

build: generate
	go build -o example .

generate:
	go generate ./...

clean:
	rm -f example
