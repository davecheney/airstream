// airstream is a network pipe that automatically discovers its peer via multicast DNS.
//
// Usage:
//
// On one machine run
//
//     echo "hello world" | airstream
//
// On another run
//
//     airstream
//
// If the two machines are on the same mdns domain the second one will now print
//
//     hello world
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/davecheney/airstream/internal/mdns"
	"github.com/davecheney/airstream/internal/netaddr"
	"github.com/miekg/dns"
)

const name = "airstream-global.local."

func serve(l net.Listener) {
	c, err := l.Accept()
	if err != nil {
		return
	}
	l.Close()
	defer c.Close()
	io.Copy(c, os.Stdin)
}

func respond(m *mdns.Mdns, l net.Listener) {
	addrs, err := netaddr.IPv4()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	rr := []dns.RR{}
	for _, a := range addrs {
		rr = append(rr, &dns.SRV{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    5,
			},
			Port:   uint16(l.Addr().(*net.TCPAddr).Port),
			Target: a.(*net.IPNet).IP.String() + ".",
		})
	}
	if err := m.Respond(rr...); err != nil {
		log.Fatalf("%+v", err)
	}
}

func main() {
	l, err := net.Listen("tcp4", ":0")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", l.Addr())

	pipe := func(r io.Reader) {
		l.Close() // close incoming connection as we are calling
		io.Copy(os.Stdout, r)

		os.Exit(0)
	}

	go serve(l)

	mdns := mdns.Mdns{
		Query: func(mdns *mdns.Mdns, addr *net.UDPAddr, q *dns.Question) {
			if q.Name != name {
				log.Println("ignoring query for", q.Name)
				return
			}
			respond(mdns, l)
		},
		Response: func(mdns *mdns.Mdns, addr *net.UDPAddr, a dns.RR) {
			switch a := a.(type) {
			case *dns.SRV:
				if a.Hdr.Name != name {
					log.Println("ignoring answer for", a.Hdr.Name)
					return
				}
				raddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", a.Target[:len(a.Target)-1], a.Port))
				if err != nil {
					log.Fatalf("%+v", err)
				}
				if raddr.Port == l.Addr().(*net.TCPAddr).Port {
					log.Println("ignoring ourselves", raddr.String())
					return
				}
				c, err := net.DialTCP("tcp", nil, raddr)
				if err != nil {
					log.Println("%+v", err)
					return
				}
				defer c.Close()
				log.Println("opening pipe to", c.RemoteAddr())
				pipe(c)
			default:
			}
		},
	}

	go func() {
		for {
			mdns.Send(dns.Question{
				Name:   name,
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			})
			time.Sleep(2 * time.Second)
		}
	}()

	if err := mdns.Listen(); err != nil {
		log.Fatalf("%+v", err)
	}
}
