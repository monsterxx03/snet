package dns

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"snet/bloomfilter"
	"snet/cache"
	"snet/cidradix"
	"snet/logger"
	"snet/utils"
)

const (
	dnsPort              = 53
	dnsTimeout           = 5
	cacheSize            = 5000
	defaultTTL           = 300 // used to cache empty A records
	bloomfilterErrorRate = 0.00001
)

type DNS struct {
	udpAddr              *net.UDPAddr
	udpListener          *net.UDPConn
	cnDNS                string
	fqDNS                string
	enforceTTL           uint32
	disableQTypes        []string
	forceFQ              []string
	hostMap              map[string]string
	blockHostsBF         *bloomfilter.Bloomfilter
	blockHosts           []string
	additionalBlockHosts []string
	chnroutesTree        *cidradix.Tree
	cache                *cache.LRU
	prefetchEnable       bool
	prefetchCount        int
	prefetchInterval     int
	l                    *logger.Logger
}

func NewServer(laddr, cnDNS, fqDNS string, enableCache bool, enforceTTL uint32, DisableQTypes []string,
	ForceFq []string, HostMap map[string]string, BlockHostFile string, AdditionalBlockHosts []string,
	prefetchEnable bool, prefetchCount int, prefetchInterval int,
	chnroutes []string, l *logger.Logger) (*DNS, error) {
	uaddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	var c *cache.LRU
	if enableCache {
		c, err = cache.NewLRU(cacheSize)
		if err != nil {
			return nil, err
		}
	}
	var bf *bloomfilter.Bloomfilter
	lines := []string{}
	if BlockHostFile != "" {
		f, err := os.Open(BlockHostFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		now := time.Now()
		for scanner.Scan() {
			l := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(l, "#") {
				continue
			}
			lines = append(lines, l)
		}
		bf, err = bloomfilter.NewBloomfilter(len(lines), bloomfilterErrorRate)
		if err != nil {
			return nil, err
		}
		// init bf
		for _, line := range lines {
			bf.Add([]byte(line))
		}
		l.Info("Load ad hosts file, cost:", time.Now().Sub(now))
	}

	// build radix tree for cidr
	tree := cidradix.NewTree()
	for _, route := range chnroutes {
		_, ipnet, err := net.ParseCIDR(route)
		if err != nil {
			return nil, err
		}
		tree.AddCIDR(ipnet)
	}
	return &DNS{
		udpAddr:              uaddr,
		cnDNS:                cnDNS,
		fqDNS:                fqDNS,
		enforceTTL:           enforceTTL,
		disableQTypes:        DisableQTypes,
		forceFQ:              ForceFq,
		hostMap:              HostMap,
		blockHostsBF:         bf,
		blockHosts:           lines,
		additionalBlockHosts: AdditionalBlockHosts,
		chnroutesTree:        tree,
		cache:                c,
		prefetchEnable:       prefetchEnable,
		prefetchCount:        prefetchCount,
		prefetchInterval:     prefetchInterval,
		l:                    l,
	}, nil
}

func (s *DNS) Run() error {
	var err error
	s.udpListener, err = net.ListenUDP("udp", s.udpAddr)
	if err != nil {
		return err
	}
	s.l.Info("DNS server listen on udp:", s.udpAddr)
	defer s.udpListener.Close()
	if s.cache != nil && s.prefetchEnable {
		s.l.Info("Starting dns prefetch ticker")
		go s.prefetchTicker()
	}
	for {
		b := make([]byte, 1024)
		n, uaddr, err := s.udpListener.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go func(uaddr *net.UDPAddr, data []byte) {
			err := s.handle(uaddr, data)
			if err != nil {
				s.l.Error(err)
			}
		}(uaddr, b[:n])
	}
}

func (s *DNS) Shutdown() error {
	if err := s.udpListener.Close(); err != nil {
		return err
	}
	return nil
}

func (s *DNS) badDomain(domain string) bool {
	if utils.DomainMatch(domain, s.additionalBlockHosts) {
		return true
	}
	// For good domain, bloomfilter can reduce lookup time
	// from 80us -> 1us. For bad domain, lookup time will increase
	// about 1us, worth the effort.
	if s.blockHostsBF != nil && s.blockHostsBF.Has([]byte(domain)) {
		// fallback to full scan, since bloomfilter has error rate
		for _, host := range s.blockHosts {
			if host == domain {
				return true
			}
		}
	}
	return false
}

func (s *DNS) isCNIP(ip net.IP) bool {
	// radix tree cost ~20us to check ip in cidr.
	// loop over whole cidr check cost 130us+,
	// worth the effort.
	if s.chnroutesTree.Contains(ip) {
		return true
	}
	return false
}

func (s *DNS) handle(reqUaddr *net.UDPAddr, data []byte) error {
	defer func() {
		if r := recover(); r != nil {
			s.l.Error("Recoverd in dns handle:\n", string(debug.Stack()))
			s.l.Info("error:", r, "data:", data)
		}
	}()

	dnsQuery, err := s.parse(data)
	if err != nil {
		return err
	}
	for _, t := range s.disableQTypes {
		if strings.ToLower(t) == strings.ToLower(dnsQuery.QType.String()) {
			resp := GetEmptyDNSResp(data)
			if _, err := s.udpListener.WriteToUDP(resp, reqUaddr); err != nil {
				return err
			}
			return nil
		}
	}
	if ip, ok := s.hostMap[dnsQuery.QDomain]; ok {
		resp := GetDNSResp(data, dnsQuery.QDomain, ip)
		if _, err := s.udpListener.WriteToUDP(resp, reqUaddr); err != nil {
			return err
		}
		return nil
	}

	if s.badDomain(dnsQuery.QDomain) {
		s.l.Debug("block ad host", dnsQuery.QDomain)
		// return 127.0.0.1 for this host
		resp := GetDNSResp(data, dnsQuery.QDomain, "127.0.0.1")
		if _, err := s.udpListener.WriteToUDP(resp, reqUaddr); err != nil {
			return err
		}
		return nil
	}
	if s.cache != nil {
		cachedData := s.cache.Get(dnsQuery.CacheKey())
		if cachedData != nil {
			s.l.Debug("dns cache hit:", dnsQuery.QDomain)
			resp := cachedData.([]byte)
			if len(resp) <= 2 {
				s.l.Error("invalid cached data", resp, dnsQuery.QDomain, dnsQuery.QType.String())
			} else {
				// rewrite first 2 bytes(dns id)
				resp[0] = data[0]
				resp[1] = data[1]
				if _, err := s.udpListener.WriteToUDP(resp, reqUaddr); err != nil {
					return err
				}
				return nil
			}
		}
	}
	raw, msg, err := s.doQuery(data, dnsQuery)
	if err != nil {
		return err
	}

	if _, err := s.udpListener.WriteToUDP(raw, reqUaddr); err != nil {
		return err
	}
	if s.cache != nil && len(raw) > 0 {
		ttl := s.getCacheTime(msg)
		// add to dns cache
		s.cache.Add(dnsQuery.CacheKey(), raw, ttl)
	}

	return nil
}

func (s *DNS) getCacheTime(msg *DNSMsg) time.Duration {
	if s.enforceTTL > 0 {
		return time.Duration(s.enforceTTL) * time.Second
	}
	if msg != nil && len(msg.ARecords) > 0 {
		return time.Duration(msg.ARecords[0].TTL) * time.Second
	}
	return defaultTTL * time.Second
}

func (s *DNS) parse(data []byte) (*DNSMsg, error) {
	msg, err := NewDNSMsg(data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *DNS) doQuery(data []byte, dnsQuery *DNSMsg) (raw []byte, msg *DNSMsg, err error) {
	var wg sync.WaitGroup
	var cnData, fqData []byte
	var cnMsg, fqMsg *DNSMsg
	wg.Add(1)
	go func(data []byte) {
		defer wg.Done()
		var err error
		fqData, err = s.queryFQ(data)
		if err != nil {
			s.l.Error("failed to query fq dns:", dnsQuery, err)
			return
		}
		fqMsg, err = s.parse(fqData)
		if err != nil {
			s.l.Error("failed to parse resp from fq dns:", err)
		}
	}(data)
	if !utils.DomainMatch(dnsQuery.QDomain, s.forceFQ) {
		var err error
		cnData, err = s.queryCN(data)
		if err != nil {
			s.l.Error("failed to query CN dns:", dnsQuery, err)
			return nil, nil, err
		}
		cnMsg, err = s.parse(cnData)
		if err != nil {
			s.l.Error("failed to parse resp from cn dns:", err)
			return nil, nil, err
		}
		if len(cnMsg.ARecords) >= 1 && s.isCNIP(cnMsg.ARecords[0].IP) {
			// if cn dns have response and it's an cn ip, we think it's a site in China
			raw = cnData
			msg = cnMsg
		} else {
			wg.Wait()
			// use fq dns's response for all ip outside of China
			raw = fqData
			msg = fqMsg
		}
	} else {
		s.l.Debug("skip cn-dns for", dnsQuery.QDomain)
		wg.Wait()
		raw = fqData
		msg = fqMsg
	}
	return
}

func (s *DNS) queryCN(data []byte) ([]byte, error) {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", s.cnDNS, dnsPort))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(dnsTimeout * time.Second)); err != nil {
		return nil, err
	}
	if _, err = conn.Write(data); err != nil {
		return nil, err
	}
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		return nil, err
	}
	return b[0:n], nil
}

func (s *DNS) queryFQ(data []byte) ([]byte, error) {
	// query fq dns by tcp, it will be captured by iptables and go out through ss
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", s.fqDNS, dnsPort))
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(dnsTimeout * time.Second)); err != nil {
		return nil, err
	}
	defer conn.Close()
	b := make([]byte, 2) // used to hold dns data length
	binary.BigEndian.PutUint16(b, uint16(len(data)))
	if _, err = conn.Write(append(b, data...)); err != nil {
		return nil, err
	}
	b = make([]byte, 2)
	if _, err = conn.Read(b); err != nil {
		return nil, err
	}

	_len := binary.BigEndian.Uint16(b)
	b = make([]byte, _len)
	if _, err = conn.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func decodeCacheKey(key string) (qdomain string, qtype RType) {
	r := strings.Split(key, ":")
	qdomain = r[0]
	_qtype, _ := strconv.Atoi(r[1])
	qtype = RType(_qtype)
	return
}

func (s *DNS) prefetchTicker() {
	ticker := time.NewTicker(time.Duration(s.prefetchInterval) * time.Second)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		s.l.Info("starting dns prefetch for top", s.prefetchCount)
		for _, item := range s.cache.PrefetchTopN(s.prefetchCount) {
			s.l.Info("prefetch for ", item.Key)
			qdomain, qtype := decodeCacheKey(item.Key)
			qdata := GetDNSQuery(qdomain, qtype)
			qmsg, err := s.parse(qdata)
			if err != nil {
				s.l.Error(err)
				continue
			}
			raw, msg, err := s.doQuery(qdata, qmsg)
			if err != nil {
				s.l.Error(err)
				continue
			}
			if len(raw) > 0 {
				ttl := s.getCacheTime(msg)
				s.cache.Evict(qmsg.CacheKey())
				s.cache.Add(qmsg.CacheKey(), raw, ttl)
			}
		}
	}
}
