[![Build Status](https://travis-ci.com/monsterxx03/snet.svg?branch=master)](https://travis-ci.com/monsterxx03/snet)

# SNET

It's a solution like: (redsocks + ss-local)/ss-redir + ChinaDNS. But all in one binary, don't depend on dnsmasq.


## Features

- SS/http-tunnel as upstream server
- Sytemwide tcp proxy (via iptables redirect) on linux desktop/server
- Works on openwrt router.
- Bypass traffic in China
- Handle DNS in the way like ChinaDNS, so website have CDN out of China won't be redirected to their overseas site.
- Local DNS cache based on TTL.

## Limation:

- linux only (tested on ubuntu 18.04 & manjaro && openwrt)
- tcp only (but dns is handled)
- ipv4 only

## Usage

Ensure **iptables** and **ipset** installed in your system.

Example config.json:

    {
        "listen-host": "127.0.0.1",
        "listen-port": 1111,
        "proxy-type": "ss",
        "proxy-timeout":  5,
        "http-proxy-host": "",
        "http-proxy-port": 8080,
        "http-proxy-auth-user": "",
        "http-proxy-auth-password": "",
        "ss-host": "ss.example.com",
        "ss-port": 8080,
        "ss-chpier-method": "aes-256-cfb",
        "ss-passwd": "passwd",
        "cn-dns": "114.114.114.114",  # dns in China
        "fq-dns": "8.8.8.8",  # clean dns out of China
        "enable-dns-cache": true,
        "enforce-ttl": 0,  # if > 0, will use this value otherthan A record's TTL
        "mode": "local" 
    }

supported proxy-type:

- ss: use ss as upstream server
- http: use http proxy server as upstream server(should support `CONNECT` method, eg: squid)

Since `snet` will modify iptables, root privilege is required. 

`sudo ./snet -config config.json`

test:

- go to `whatsmyip.com`, ip should be your ss server ip.
- go to `myip.ipip.net`, ip should be your local ip in China.

If you want to use it on openwrt, change `mode` to `router`, and listen-host should be your router's ip or `0.0.0.0`

## Notice

If crash or force killed(kill -9), snet will have no chance to cleanup iptables, it will make you have no internet access.

You need to clean them manually(or restart snet, it will try to cleanup).

Way 1:

    sudo ./snet -clean -v

Way 2 (if you're sure no useful iptable rules on your system):

    sudo iptables -t nat -F  
    # if you install docker, docker's iptable rules will be flushed as well, just restart docker it will recreate them.

## Known issue:

- Manjaro's NetworkManager will create a ipv6 dns nameserver in /etc/resolv.conf, eg: `nameserver fe80::1%enp51s0`.
If it's first nameserver, dns query will bypass `snet`(since I didn't handle ipv6), you need to disable ipv6 or put it on second line.
- Chrome's cache for google.com is wired.If you can visit youtube.com or twitter.com, but can't open google.com, try to restart chrome to clean dns cache.
- cn-dns should be different with the one in your /et/resolv.conf, otherwise dns lookup will by pass snet (iptable rules in SNET chain)

## TODO:

- Filter by domain.
- Stats api
