package dns

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/brunexgeek/hugye/pkg/binary"
	"github.com/brunexgeek/hugye/pkg/dfa"
)

type Resolver struct {
	id     uint16
	extdns []ExternalDNS
	defdns *ExternalDNS
	conn   *net.UDPConn
}

type ExternalDNS struct {
	Address *net.UDPAddr
	Name    string
	Targets *dfa.Tree
}

type Ticket struct {
	Id   uint16
	Conn *net.UDPConn
}

var last_port int = 63009

func NewResolver(extdns []ExternalDNS) (*Resolver, error) {
	result := &Resolver{extdns: extdns}
	last_port++
	addr, err := netip.ParseAddrPort(fmt.Sprintf("0.0.0.0:%d", last_port))
	if err != nil {
		return nil, err
	}
	result.conn, err = net.ListenUDP("udp4", net.UDPAddrFromAddrPort(addr))
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(result.extdns); i++ {
		item := &result.extdns[i]
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

	var addr = r.defdns.Address
	if len(host) > 0 {
		for _, item := range r.extdns {
			if item.Targets != nil && item.Targets.Match(host) {
				addr = item.Address
				break
			}
		}
	}

	// send query
	size, err := r.conn.WriteToUDP(buf, addr)

	// recover the original ID
	binary.Write16(buf, 0, oid)

	if err != nil {
		return nil, err
	} else if size != len(buf) {
		return nil, fmt.Errorf("Unable to send all data")
	}

	return &Ticket{Id: id, Conn: r.conn}, nil
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
