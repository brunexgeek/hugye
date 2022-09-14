package main

import (
	"fmt"
	"net"
	"time"
)

type Resolver struct {
	conn *net.UDPConn
	id   uint16
}

func MakeResolver(extdns *net.UDPAddr) (*Resolver, error) {
	conn, err := net.DialUDP("udp4", nil, extdns)
	if err != nil {
		return nil, err
	}
	return &Resolver{conn: conn, id: 0}, nil
}

func (r *Resolver) Send(buf []byte, id uint16) (uint16, error) {
	// replace the current ID
	var oid uint16
	read_u16(buf, 0, &oid)
	write_u16(buf, 0, id)

	// send query
	size, err := r.conn.Write(buf)

	// recover the original ID
	write_u16(buf, 0, oid)

	if err != nil {
		return 0, err
	} else if size != len(buf) {
		return 0, fmt.Errorf("Unable to send all data")
	}

	return id, nil
}

func (r *Resolver) Receive(timeout int) ([]byte, error) {
	r.conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
	buf := make([]byte, 1024)
	size, err := r.conn.Read(buf)
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
