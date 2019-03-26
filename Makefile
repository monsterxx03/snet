all: build 

build:
	go generate
	go fmt
	go build -o bin/snet

run:
	sudo ./bin/snet -ss-passwd abc

update:
	curl http://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest -o apnic.txt
	go generate
	go fmt

clear:
	sudo iptables -t nat -F
	sudo iptables -t nat -X BYPASS_SNET
