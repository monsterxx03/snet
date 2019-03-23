all: build run

build:
	go build -o bin/snet

run:
	sudo iptables -t nat -N SNET
	sudo iptables -t nat -A SNET -d x.x.x.x/32 -j RETURN
	sudo iptables -t nat -A SNET -d 0.0.0.0/8 -j RETURN
	sudo iptables -t nat -A SNET -d 10.0.0.0/8 -j RETURN
	sudo iptables -t nat -A SNET -d 100.64.0.0/108 -j RETURN
	sudo iptables -t nat -A SNET -d 127.0.0.0/8 -j RETURN
	sudo iptables -t nat -A SNET -d 169.254.0.0/16 -j RETURN
	sudo iptables -t nat -A SNET -d 172.16.0.0/12 -j RETURN
	sudo iptables -t nat -A SNET -d 192.0.2.0/24 -j RETURN
	sudo iptables -t nat -A SNET -d 192.88.99.0/24 -j RETURN
	sudo iptables -t nat -A SNET -d 192.168.0.0/16 -j RETURN
	sudo iptables -t nat -A SNET -d 192.18.0.0/15 -j RETURN
	sudo iptables -t nat -A SNET -d 192.51.100.0/24 -j RETURN
	sudo iptables -t nat -A SNET -d 203.0.113.0/24 -j RETURN
	sudo iptables -t nat -A SNET -d 224.0.0.0/4 -j RETURN
	sudo iptables -t nat -A SNET -d 240.0.0.0/4 -j RETURN
	sudo iptables -t nat -A SNET -d 255.255.255.255/32 -j RETURN
	sudo iptables -t nat -A SNET -p tcp -j REDIRECT --to-ports 1111
	sudo iptables -t nat -A OUTPUT -p tcp  -j SNET
	./bin/snet

clear:
	sudo iptables -t nat -F
	sudo iptables -t nat -X SNET
