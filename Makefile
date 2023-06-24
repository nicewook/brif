
.PHONY: run build install

run:
	@go run .

dev:
	@RUN_MODE=dev go run .

build:
	go build -o brif

install:
	go install