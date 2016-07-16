package mdns

// Advertise network services via multicast DNS

import (
	"net"

	//"github.com/miekg/dns"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Mdns describes a mdns listener.
type Mdns struct {

	// Query is called for each Question message received.
	Query func(*Mdns, *net.UDPAddr, *dns.Question)

	// Response is called for each Answer message received.
	Response func(*Mdns, *net.UDPAddr, dns.RR)
}

// Listen listens for packets on the default IPv4 mulitcast address.
// Listen blocks until an error is returned from the underlying socket.
func (m *Mdns) Listen() error {
	c, err := listen()
	if err != nil {
		return err
	}
	defer c.Close()
	for {
		msg, addr, err := read(c)
		if err != nil {
			return err
		}

		if m.Query != nil {
			for _, q := range msg.Question {
				m.Query(m, addr, &q)
			}
		}

		if m.Response != nil {
			for _, a := range msg.Answer {
				m.Response(m, addr, a)
			}
		}
	}
}

func (m *Mdns) Send(questions ...dns.Question) error {
	msg := &dns.Msg{
		Question: questions,
	}
	return m.send(msg)
}

func (m *Mdns) Respond(answers ...dns.RR) error {
	msg := &dns.Msg{
		Answer: answers,
	}
	return m.send(msg)
}

func (m *Mdns) send(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return errors.WithStack(err)
	}
	c, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	defer c.Close()
	_, err = c.Write(buf)
	return errors.WithStack(err)
}

func listen() (*net.UDPConn, error) {
	ipv4mcastaddr := &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}
	l, err := net.ListenMulticastUDP("udp4", nil, ipv4mcastaddr)
	return l, errors.WithStack(err)
}

// read reads one mdns packet from the write and decodes it.
func read(c *net.UDPConn) (*dns.Msg, *net.UDPAddr, error) {
	buf := make([]byte, 1500)
	read, addr, err := c.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	var msg dns.Msg
	err = msg.Unpack(buf[:read])
	return &msg, addr, errors.WithStack(err)
}
