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

func preprocess_job(ctx *WorkerContext, job *Job) (*Job, error) {
	ValidateMessage(job.request.bytes)

	// TODO: try cache

	// send query to external DNS
	id, error := ctx.resolver.Send(job.request.bytes, ctx.resolver.NextId())
	if error != nil {
		return nil, error
	}
	job.id = id
	return job, nil
}

func worker(ctx *WorkerContext) error {
	wait_list := make(map[uint16]*Job)

	for running {
		//
		// Accept new jobs
		//
		{
			var timeout time.Duration = 500
			if len(wait_list) > 0 {
				timeout = 5
			}
			select {
			case job := <-ctx.input:
				job, err := preprocess_job(ctx, job)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if job != nil {
					fmt.Printf("Job #%d accepted\n", job.id)
					wait_list[job.id] = job
				}
			case <-time.After(timeout * time.Millisecond):

			}
		}

		//
		// Process each UDP response
		//
		for true {
			//fmt.Println("Checking for responses")
			buffer, err := ctx.resolver.Receive(1)
			if err != nil {
				//fmt.Println(err)
				break
			}

			var id uint16
			_, err = read_u16(buffer, 0, &id)
			if err != nil {
				continue
			}
			fmt.Printf("-- Found response #%d", id)

			job := wait_list[id]
			if job != nil {
				job.response.bytes = buffer
				// replace the ID
				write_u16(job.response.bytes, 0, job.request.oid)
				// send response to client
				size, err := job.conn.WriteTo(job.response.bytes, job.request.remote)
				if err != nil {
					fmt.Println(err)
				} else if size != len(job.response.bytes) {
					fmt.Println("Unable to send all data")
				}
				delete(wait_list, id)
			} else {
				fmt.Printf("-- Job #%d not in waiting list", id)
			}
		}

		//
		// Check jobs in the waiting list to finish or discard them.
		//
	}
	return fmt.Errorf("Worker done")
}

func MakeJob(conn *net.UDPConn, addr *net.UDPAddr, buf []byte, size int) *Job {
	job := Job{}
	job.conn = conn
	job.start_time = time.Now().UnixMilli()
	job.request.bytes = buf[:size]
	job.request.remote = addr

	var oid uint16
	read_u16(job.request.bytes, 0, &oid)
	job.request.oid = oid

	return &job
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
	worker, err := processor.StartWorker(worker)
	if worker == nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Worker started")

	for running {
		conn.SetDeadline(time.Now().Add(dur))
		size, remote, error := conn.ReadFromUDP(buffer)
		if error != nil {
			//fmt.Println(error)
			continue
		}
		fmt.Println("Job found")
		job := MakeJob(conn, remote, buffer, size)
		worker.GetContext().input <- job
		fmt.Println("Job sent")
	}
	processor.Await()
}
