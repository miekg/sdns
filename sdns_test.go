package main

import (
	"net"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"github.com/semihalev/log"
	"github.com/stretchr/testify/assert"
	"github.com/yl2chen/cidranger"
)

const (
	testDomain = "www.google.com"
)

var (
	ginr *gin.Engine
)

func TestMain(m *testing.M) {
	log.Root().SetHandler(log.LvlFilterHandler(0, log.StdoutHandler))

	Config.Maxdepth = 30
	Config.Interval = 200
	Config.Nullroute = "0.0.0.0"
	Config.Nullroutev6 = "0:0:0:0:0:0:0:0"
	Config.Bind = ":0"
	Config.BindTLS = ""
	Config.BindDOH = ""
	Config.API = ""

	accessList = cidranger.NewPCTrieRanger()
	_, ipnet, _ := net.ParseCIDR("0.0.0.0/0")
	accessList.Insert(cidranger.NewBasicRangerEntry(*ipnet))
	_, ipnet, _ = net.ParseCIDR("::0/0")
	accessList.Insert(cidranger.NewBasicRangerEntry(*ipnet))

	gin.SetMode(gin.TestMode)
	ginr = gin.New()

	block := ginr.Group("/api/v1/block")
	{
		block.GET("/exists/:key", existsBlock)
		block.GET("/get/:key", getBlock)
		block.GET("/remove/:key", removeBlock)
		block.GET("/set/:key", setBlock)
	}

	m.Run()
}

func Test_SDNS(t *testing.T) {
	Config.Bind = ":0"
	Config.BindTLS = ""
	Config.BindDOH = ""
	Config.API = ""
	Config.Timeout = 15

	startSDNS()

	time.Sleep(2 * time.Second)
}

func BenchmarkResolver(b *testing.B) {
	s, addrstr, err := RunLocalUDPServer("127.0.0.1:0")
	assert.NoError(b, err)

	defer s.Shutdown()

	req := new(dns.Msg)
	req.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)
	req.RecursionDesired = true

	c := new(dns.Client)

	//caching
	_, _, err = c.Exchange(req, addrstr)
	assert.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Exchange(req, addrstr)
	}
}

func BenchmarkUDPHandler(b *testing.B) {
	h := NewHandler()

	req := new(dns.Msg)
	req.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)
	req.RecursionDesired = true
	req.SetEdns0(DefaultMsgSize, true)

	//caching
	resp := h.query("udp", req)
	assert.NotNil(b, resp)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.query("udp", req)
	}
}

func BenchmarkTCPHandler(b *testing.B) {
	h := NewHandler()

	req := new(dns.Msg)
	req.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)
	req.RecursionDesired = true

	//caching
	resp := h.query("tcp", req)
	assert.NotNil(b, resp)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.query("tcp", req)
	}
}
