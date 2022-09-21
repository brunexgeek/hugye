package dns

import (
	"fmt"
	"net"
	"time"

	"github.com/brunexgeek/hugye/pkg/binary"
	"github.com/brunexgeek/hugye/pkg/dfa"
)

type Resolver struct {
	id     uint16
	extdns []ExternalDNS
	defdns *ExternalDNS
}

type ExternalDNS struct {
	Address *net.UDPAddr
	Name    string
	Targets *dfa.Tree
	conn    *net.UDPConn
}

type Ticket struct {
	Id   uint16
	Conn *net.UDPConn
}

func NewResolver(extdns []ExternalDNS) (*Resolver, error) {
	result := &Resolver{extdns: extdns}
	var err error = nil
	for i := 0; i < len(result.extdns); i++ {
		item := &result.extdns[i]
		item.conn, err = net.DialUDP("udp4", nil, item.Address)
		if err != nil {
			return nil, err
		}
		if item.Targets == nil {
			result.defdns = item
		}
	}
	if result.defdns == nil {
		return nil, fmt.Errorf("Missing default external DNS")
	}
	return result, nil
}

func (r *Resolver) Send(host string, buf []byte, id uint16) (*Ticket, error) {
	// replace the current ID
	var oid uint16
	binary.Read16(buf, 0, &oid)
	binary.Write16(buf, 0, id)

	var conn *net.UDPConn = r.defdns.conn
	if len(host) > 0 {
		for _, item := range r.extdns {
			if item.Targets != nil && item.Targets.Match(host) {
				conn = item.conn
				break
			}
		}
	}

	// send query
	size, err := conn.Write(buf)

	// recover the original ID
	binary.Write16(buf, 0, oid)

	if err != nil {
		return nil, err
	} else if size != len(buf) {
		return nil, fmt.Errorf("Unable to send all data")
	}

	return &Ticket{Id: id, Conn: conn}, nil
}

func (r *Resolver) Receive(ticket *Ticket, timeout int) ([]byte, error) {
	if ticket == nil {
		return nil, fmt.Errorf("Invalid ticket")
	}

	ticket.Conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
	buf := make([]byte, 1024)
	size, err := ticket.Conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:size], nil
}

func (r *Resolver) NextId() uint16 {
	r.id++
	if r.id == 0 {
		r.id++
	}
	return r.id
}
