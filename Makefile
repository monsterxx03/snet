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

test:
	go test

build_hiwifi:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build
