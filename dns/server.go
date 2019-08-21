package dns

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"snet/bloomfilter"
	"snet/cache"
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
	chnroutes            []*net.IPNet
	cache                *cache.LRU
	l                    *logger.Logger
}

func NewServer(laddr, cnDNS, fqDNS string, enableCache bool, enforceTTL uint32, DisableQTypes []string, ForceFq []string, HostMap map[string]string, BlockHostFile string, AdditionalBlockHosts, chnroutes []string, l *logger.Logger) (*DNS, error) {
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
		for scanner.Scan() {
			lines = append(lines, strings.TrimSpace(scanner.Text()))
		}
		bf, err = bloomfilter.NewBloomfilter(len(lines), bloomfilterErrorRate)
		if err != nil {
			return nil, err
		}
		// init bf
		for _, line := range lines {
			bf.Add([]byte(line))
		}
	}

	cnRoutes := make([]*net.IPNet, 0, len(chnroutes))
	for _, route := range chnroutes {
		_, ipnet, err := net.ParseCIDR(route)
		if err != nil {
			return nil, err
		}
		cnRoutes = append(cnRoutes, ipnet)
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
		chnroutes:            cnRoutes,
		cache:                c,
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
	for _, ipnet := range s.chnroutes {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *DNS) handle(reqUaddr *net.UDPAddr, data []byte) error {
	defer func() {
		if r := recover(); r != nil {
			s.l.Error("Recoverd in dns handle:\n", string(debug.Stack()))
			s.l.Info(data)
		}
	}()

	var wg sync.WaitGroup
	var cnData, fqData []byte
	var cnMsg, fqMsg *DNSMsg
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
		cachedData := s.cache.Get(fmt.Sprintf("%s:%s", dnsQuery.QDomain, dnsQuery.QType))
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
	if !utils.DomainMatch(dnsQuery.QDomain, s.forceFQ) {
		wg.Add(1)
		go func(data []byte) {
			defer wg.Done()
			var err error
			cnData, err = s.queryCN(data)
			if err != nil {
				s.l.Error("failed to query CN dns:", dnsQuery, err)
			}
		}(data)
	} else {
		s.l.Debug("skip cn-dns for", dnsQuery.QDomain)
	}
	wg.Add(1)
	go func(data []byte) {
		defer wg.Done()
		var err error
		fqData, err = s.queryFQ(data)
		if err != nil {
			s.l.Error("failed to query fq dns:", dnsQuery, err)
		}
	}(data)

	wg.Wait()

	if len(cnData) > 0 {
		cnMsg, err = s.parse(cnData)
		if err != nil {
			return err
		}
	}
	if len(fqData) > 0 {
		fqMsg, err = s.parse(fqData)
		if err != nil {
			return err
		}
	}
	var raw []byte
	useMsg := cnMsg
	if cnMsg != nil && len(cnMsg.ARecords) >= 1 && s.isCNIP(cnMsg.ARecords[0].IP) {
		// if cn dns have response and it's an cn ip, we think it's a site in China
		raw = cnData
	} else {
		// use fq dns's response for all ip outside of China
		raw = fqData
		useMsg = fqMsg
	}
	if _, err := s.udpListener.WriteToUDP(raw, reqUaddr); err != nil {
		return err
	}
	if s.cache != nil && len(raw) > 0 {
		var ttl uint32
		if s.enforceTTL > 0 {
			ttl = s.enforceTTL
		} else {
			// if enforceTTL not set, follow A record's TTL
			if useMsg != nil && len(useMsg.ARecords) > 0 {
				ttl = useMsg.ARecords[0].TTL
			} else {
				ttl = defaultTTL
			}
		}
		// add to dns cache
		s.cache.Add(fmt.Sprintf("%s:%s", dnsQuery.QDomain, dnsQuery.QType), raw, time.Now().Add(time.Second*time.Duration(ttl)))
	}

	return nil
}

func (s *DNS) parse(data []byte) (*DNSMsg, error) {
	msg, err := NewDNSMsg(data)
	if err != nil {
		return nil, err
	}
	return msg, nil
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
