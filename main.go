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
	return nil
}

func worker(ctx *WorkerContext) error {
	fmt.Println(ctx)
	for running {
		select {
		case job := <-ctx.input:
			process_job(ctx, &job)
			fmt.Println("Get job ", job)
		case <-time.After(4 * time.Second):
			fmt.Println("Still waiting...")
		}
	}
	return fmt.Errorf("Worker done")
}

func main() {
	address, error := netip.ParseAddrPort("127.0.0.7:5300")
	if error != nil {
		return
	}
	conn, error := net.ListenUDP("udp4", net.UDPAddrFromAddrPort(address))
	dur := time.Second * 5
	conn.SetDeadline(time.Now().Add(dur))
	buffer := make([]byte, 1024)

	processor := Processor{}
	future := processor.StartWorker(worker)
	fmt.Println("Worker started")

	for running {
		conn.SetDeadline(time.Now().Add(dur))
		size, remote, error := conn.ReadFromUDP(buffer)
		if error != nil {
			fmt.Println(error)
			continue
		}
		fmt.Println("Job found")
		job := Job{}
		job.start_time = time.Now().UnixMilli()
		job.request.bytes = buffer[:size]
		job.request.remote = remote
		future.GetContext().input <- job
		fmt.Println("Job sent")
	}
	future.Await()
}
