all: build 

LDFLAGS="-X main.sha1Ver=`git rev-parse HEAD` -X main.buildAt=`date -u +'%Y-%m-%dT%T%z'`"

build:
	go build -ldflags $(LDFLAGS) -o bin/snet

build_linux_amd64:
	GOOS=linux GOARCH=amd64  go build -ldflags $(LDFLAGS) -o bin/snet_linux_amd64


build_darwin_amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bin/snet_darwin_amd64

update:
	curl http://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest -o apnic.txt
	curl https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts -o ad_hosts.txt
	go generate && curl https://raw.githubusercontent.com/privacy-protection-tools/anti-AD/master/anti-ad-domains.txt >> ad_hosts 
	go fmt

test:
	go test --race -v $$(go list ./...| grep -v -e /vendor/)

build_hiwifi:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags $(LDFLAGS) -o bin/snet_mipsle_softfloat

build_erx:
	GOOS=linux GOARCH=mipsle go build -ldflags $(LDFLAGS) -o bin/snet_mipsle

deb:
	cp config.json.example debain/etc/snet/config.json
	cp bin/snet debain/usr/local/bin/snet
	dpkg -b debain snet.deb
