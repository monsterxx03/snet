all: build 

build:
	go build -o bin/snet

run:
	sudo ./bin/snet -ss-passwd abc

clear:
	sudo iptables -t nat -F
	sudo iptables -t nat -X SNET
