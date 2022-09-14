package main

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"time"
)

var running = true

func install_signal_hook() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if running {
			running = false
		} else {
			os.Exit(1)
		}
	}()
}

func main() {
	install_signal_hook()
	address, error := netip.ParseAddrPort("127.0.0.7:5300")
	if error != nil {
		return
	}
	extdns, error := netip.ParseAddrPort("8.8.8.8:53")
	if error != nil {
		return
	}
	conn, error := net.ListenUDP("udp4", net.UDPAddrFromAddrPort(address))
	dur := time.Second * 1
	conn.SetDeadline(time.Now().Add(dur))
	buffer := make([]byte, 1024)

	processor := Processor{extdns: net.UDPAddrFromAddrPort(extdns)}
	worker, err := processor.StartWorker()
	if worker == nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Worker started")

	for running {
		conn.SetDeadline(time.Now().Add(dur))
		_, _, error := conn.ReadFromUDP(buffer)
		if error != nil {
			//fmt.Println(error)
			continue
		}
		fmt.Println("Job found")
		var value string
		read_qname(buffer, 12, &value)
		fmt.Println(value)

		//job := MakeJob(conn, remote, buffer, size)
		//worker.GetContext().input <- job
		//fmt.Println("Job sent")
	}
	processor.Await()
}
