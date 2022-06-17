package udp

import (
	"log"
	"net"

	"github.com/golang/snappy"
	"github.com/net-byte/vtun/common/cipher"
	"github.com/net-byte/vtun/common/config"
	"github.com/net-byte/vtun/tun"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// StartClient starts the udp client
func StartClient(config config.Config) {
	log.Printf("vtun udp client started on %v", config.LocalAddr)
	iface := tun.CreateTun(config)
	serverAddr, err := net.ResolveUDPAddr("udp", config.ServerAddr)
	if err != nil {
		log.Fatalln("failed to resolve server addr:", err)
	}
	localAddr, err := net.ResolveUDPAddr("udp", config.LocalAddr)
	if err != nil {
		log.Fatalln("failed to get udp socket:", err)
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatalln("failed to listen on udp socket:", err)
	}
	if localAddr.IP.To4() != nil {
		p := ipv4.NewPacketConn(conn)
		if err := p.SetTOS(0xb8); err != nil { // DSCP EF
			log.Fatalln("failed to set conn tos:", err)
		}
	} else {
		p := ipv6.NewPacketConn(conn)
		if err := p.SetTrafficClass(0xb8); err != nil { // DSCP EF
			log.Fatalln("failed to set conn tos:", err)
		}
	}
	defer conn.Close()
	c := &Client{config: config, iface: iface, localConn: conn, serverAddr: serverAddr}
	go c.udpToTun()
	c.tunToUdp()
}

// The client struct
type Client struct {
	config     config.Config
	iface      *water.Interface
	localConn  *net.UDPConn
	serverAddr *net.UDPAddr
}

// udpToTun sends packets from udp to tun
func (c *Client) udpToTun() {
	packet := make([]byte, 4096)
	for {
		n, _, err := c.localConn.ReadFromUDP(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]
		if c.config.Compress {
			b, err = snappy.Decode(nil, b)
			if err != nil {
				continue
			}
		}
		if c.config.Obfs {
			b = cipher.XOR(b)
		}
		c.iface.Write(b)
	}
}

// tunToUdp sends packets from tun to udp
func (c *Client) tunToUdp() {
	packet := make([]byte, 4096)
	for {
		n, err := c.iface.Read(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]
		if c.config.Obfs {
			b = cipher.XOR(b)
		}
		if c.config.Compress {
			b = snappy.Encode(nil, b)
		}
		c.localConn.WriteToUDP(b, c.serverAddr)
	}
}
