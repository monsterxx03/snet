all: build run

build:
	go build -o bin/snet

run:
	sudo ./bin/snet

dev:
	sudo ip route add 10.100.0.0/24 via 10.100.0.1 dev tun0
