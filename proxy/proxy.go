package proxy

import (
	"errors"
	"net"
)

type Config interface{}

type Proxy interface {
	Init(c Config) error
	Dial(host string, port int) (net.Conn, error)
	Pipe(src, dst net.Conn) error
	Close() error
}

var upstreams = map[string]Proxy{}

func Register(name string, t Proxy) {
	if _, ok := upstreams[name]; !ok {
		upstreams[name] = t
	} else {
		panic("Tunnel type " + name + " already existed")
	}
}

func Get(name string) (Proxy, error) {
	if t, ok := upstreams[name]; ok {
		return t, nil
	}
	return nil, errors.New("unknow tunnel type:" + name)
}
