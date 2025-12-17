.PHONY: build test lint demo demo-build demo-run clean

build:
	go build -o snitch .

test:
	go test ./...

lint:
	golangci-lint run

demo: demo-build demo-run

demo-build:
	docker build -f demo/Dockerfile -t snitch-demo .

demo-run:
	docker run --rm -v $(PWD)/demo:/output snitch-demo

clean:
	rm -f snitch
	rm -f demo/demo.gif

