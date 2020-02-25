[![Build Status](https://travis-ci.com/monsterxx03/snet.svg?branch=master)](https://travis-ci.com/monsterxx03/snet)

# SNET

It's a solution like: (redsocks + ss-local)/ss-redir + ChinaDNS. But all in one binary, don't depend on dnsmasq.


## Features

- ss/http-tunnel/tls-tunnel/socks5 as upstream server
- Sytemwide tcp proxy on linux desktop/server(iptables), MacOS desktop (pfctl)
- Works on openwrt router
- Bypass traffic in China
- Handle DNS in the way like ChinaDNS, so website have CDN out of China won't be redirected to their overseas site
- Local DNS cache based on TTL
- block by domain name
- hostname map
- DNS prefetch

## Limation:

- tcp only (but dns is handled)
- ipv4 only

## Tested on:

Desktop:

- manjaro
- ubuntu 18.04
- MacOS 10.13.6

Router:

- hiwifi2
- ubnt er-x

## Usage

### As upstream server

Only support tls tunnel when run as upstream server

Generate tls key pair:

- openssl genrsa -out server.key 2048
- openssl req -new -x509 -key server.key -out server.pem -days 3650

Run:

    ./snet -tlsserver 0.0.0.0:9999 -tlstoken randomstringxxx  -tlskey server.key -tlscrt server.pem


### As client

For linux: ensure **iptables** and **ipset** installed in your system.

For macos: pfctl is included by default, no extra dependences.

Example config.json:

    {
        "listen-host": "127.0.0.1",
        "listen-port": 1111,
        "proxy-type": "ss",
        "proxy-timeout":  30,
        # `bypassCN` or `global`, default to `bypassCN`
        "proxy-scope": "bypassCN",
        # target host list will bypass snet
        "bypass-hosts": ["a.com"],
        # only work on "mode": "router", traffic from those ips will bypass snet, use case: home NAS
        "bypass-src-ips": ["192.168.1.100"],

        # config used when proxy-type is "http"
        "http-proxy-host": "",
        "http-proxy-port": 8080,
        "http-proxy-auth-user": "",
        "http-proxy-auth-password": "",

        # config used when proxy-type is "ss"
        "ss-host": "ss.example.com",
        "ss-port": 8080,
        # https://github.com/shadowsocks/shadowsocks-go/blob/1.2.1/shadowsocks/encrypt.go#L159
        "ss-chpier-method": "aes-256-cfb",
        "ss-passwd": "passwd",

        # config used when proxy-type is "tls"
        "tls-host": "",
        "tls-port": 443,
        "tls-token": "tlstoken",

        # config used when proxy-type is "socks5"
        "socks5-host": "",
        "socks5-port": 1080,
        "socks5-auth-user": "",
        "socks5-auth-password": "",

        "cn-dns": "114.114.114.114",  # dns in China
        "fq-dns": "8.8.8.8",  # clean dns out of China
        "enable-dns-cache": true,
        "enforce-ttl": 3600,  # if > 0, will use this value otherthan A record's TTL
        "disable-qtypes": ["AAAA"], # return empty dns msg for those query types
        "force-fq": ["*.cloudfront.net"], # domain pattern matched will skip cn-dns query

        "dns-prefetch-enable": true,
        "dns-prefetch-count":  100,  # prefetch top 10 freq used domains in cache.
        "dns-prefetch-interval": 60, 

        "host-map": {
            "google.com": "2.2.2.2"  # map host and ip
        },
        "block-host-file": "", # if set, domain name in this file will return 127.0.0.1 to client
        "block-hosts": ["*.hpplay.cn"], # support block hosts with wildcard
        "mode": "local"   # run on desktop: local, run on router: router
    }

supported proxy-type:

- ss: use ss as upstream server
- http: use http proxy server as upstream server(should support `CONNECT` method, eg: squid)
- tls: use snet tls tunnel as upstream server
- socks5: use socks5 as upstream server. Note: if your socks5 proxy server is running on same host with snet, ensure to add socks5's upstream server address to snet's `bypass-hosts` list, or socks5's traffic to upstream server will be hijacked by snet, being a loop.

Since `snet` will modify iptables/pf, root privilege is required. 

`sudo ./snet -config config.json`

Test (proxy-scope = bypassCN):

- curl `ifconfig.me`, ip should be your ss server ip.
- curl `myip.ipip.net`, ip should be your local ip in China.

If proxy-scope is `global`, both should return ss server ip.

If you use it on router, change `mode` to `router`, and listen-host should be your router's ip or `0.0.0.0`


## For traffic from docker container

Change `listen-host` to `0.0.0.0`, `mode` to `router`.

Traffic from docker container's src ip is 172.17.0.1/16, it will go through `nat prerouting chain` -> `filter foward chain` -> `nat postrouting chain`, not `nat output chain`.

The main reason I need a `client mode` and `router mode` is handling DNS redirct, don't know how to make it work for `PREROUTING chain` and `OUTPUT chain` at the same time.

## Notice

If crash or force killed(kill -9), snet will have no chance to cleanup iptables/pf rules, it will make you have no internet access.

You need to clean them manually(If restart snet, it will try to cleanup) or restart your laptop :(

Linux:

    sudo iptables -t nat -F  
    # if you install docker, docker's iptable rules will be flushed as well, just restart docker it will recreate them.

MacOS:

    sudo pfctl -d

## Known issue:

- Manjaro's NetworkManager will create a ipv6 dns nameserver in /etc/resolv.conf, eg: `nameserver fe80::1%enp51s0`.
If it's first nameserver, dns query will bypass `snet`(since I didn't handle ipv6), you need to disable ipv6 or put it on second line.
- Chrome's cache for google.com is wired.If you can visit youtube.com or twitter.com, but can't open google.com, try to restart chrome to clean dns cache.
- cn-dns should be different with the one in your /et/resolv.conf, otherwise dns lookup will by pass snet (iptable rules in SNET chain)

## artilces:

- https://blog.monsterxx03.com/tags/snet/ 
