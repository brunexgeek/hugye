package dns

import (
	"fmt"
	"net"
	"time"

	"github.com/brunexgeek/hugye/pkg/binary"
	"github.com/brunexgeek/hugye/pkg/domain"
)

type resolver struct {
	conn *net.UDPConn
	id   *uint16
}

func NewResolver(extdns *net.UDPAddr) (domain.Resolver, error) {
	conn, err := net.DialUDP("udp4", nil, extdns)
	if err != nil {
		return nil, err
	}
	return &resolver{conn: conn, id: new(uint16)}, nil
}

func (r *resolver) Send(buf []byte, id uint16) (uint16, error) {
	// replace the current ID
	var oid uint16
	binary.Read16(buf, 0, &oid)
	binary.Write16(buf, 0, id)

	// send query
	size, err := r.conn.Write(buf)

	// recover the original ID
	binary.Write16(buf, 0, oid)

	if err != nil {
		return 0, err
	} else if size != len(buf) {
		return 0, fmt.Errorf("Unable to send all data")
	}

	return id, nil
}

func (r *resolver) Receive(timeout int) ([]byte, error) {
	r.conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
	buf := make([]byte, 1024)
	size, err := r.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:size], nil
}

func (r *resolver) NextId() uint16 {
	(*r.id)++
	if *r.id == 0 {
		(*r.id)++
	}
	return *r.id
}
