all: build run

build:
	go build -o bin/snet

run:
	sudo ./bin/snet
