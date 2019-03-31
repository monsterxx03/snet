all: build 

build:
	go build -ldflags "-X main.sha1Ver=`git rev-parse HEAD` -X main.buildAt=`date -u +'%Y-%m-%dT%T%z'`" -o bin/snet

run:
	sudo ./bin/snet -ss-passwd abc

update:
	curl http://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest -o apnic.txt
	go generate
	go fmt

test:
	go get .
	go test

build_hiwifi:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build
