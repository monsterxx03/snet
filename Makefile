all: build 

build:
	go build -ldflags "-X main.sha1Ver=`git rev-parse HEAD` -X main.buildAt=`date -u +'%Y-%m-%dT%T%z'`" -o bin/snet

run:
	sudo ./bin/snet -ss-passwd abc

update:
	curl http://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest -o apnic.txt
	go generate
	go fmt

update_hosts:
	wget https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts

test:
	go test --race -v $$(go list ./...| grep -v -e /vendor/)

build_hiwifi:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build

build_erx:
	GOOS=linux GOARCH=mipsle go build

deb:
	cp config.json.example debain/etc/snet/config.json
	cp bin/snet debain/usr/local/bin/snet
	dpkg -b debain snet.deb
