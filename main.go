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

func process_job(ctx *WorkerContext, job *Job) error {
	ValidateMessage(job.request.bytes)

	conn, error := net.DialUDP("udp4", nil, ctx.processor.extdns)
	if error != nil {
		return error
	}
	size, error := conn.Write(job.request.bytes)
	if error != nil {
		return error
	} else if size < len(job.request.bytes) {
		return fmt.Errorf("Sent less bytes than expected")
	}
	fmt.Println("Message sent to external DNS")
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	size, error = conn.Read(buffer)
	if error != nil {
		return error
	}
	job.response.bytes = buffer[:size]
	fmt.Printf("Message received from external DNS (%d bytes)", size)
	size, error = job.conn.WriteToUDP(job.response.bytes, job.request.remote)
	if error != nil {
		return error
	} else if size < len(job.response.bytes) {
		return fmt.Errorf("Sent less bytes than expected")
	}
	return nil
}

func worker(ctx *WorkerContext) error {
	for running {
		select {
		case job := <-ctx.input:
			process_job(ctx, &job)
		case <-time.After(4 * time.Second):
			continue
		}
	}
	return fmt.Errorf("Worker done")
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
	dur := time.Second * 5
	conn.SetDeadline(time.Now().Add(dur))
	buffer := make([]byte, 1024)

	processor := Processor{extdns: net.UDPAddrFromAddrPort(extdns)}
	future := processor.StartWorker(worker)
	fmt.Println("Worker started")

	for running {
		conn.SetDeadline(time.Now().Add(dur))
		size, remote, error := conn.ReadFromUDP(buffer)
		if error != nil {
			//fmt.Println(error)
			continue
		}
		fmt.Println("Job found")
		job := Job{}
		job.conn = conn
		job.start_time = time.Now().UnixMilli()
		job.request.bytes = buffer[:size]
		job.request.remote = remote
		future.GetContext().input <- job
		fmt.Println("Job sent")
	}
	future.Await()
}
