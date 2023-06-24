
.PHONY run build install
run:
	go run .

build:
	go build -o brif

install:
	go install