package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/brunexgeek/hugye/pkg/binary"
	"github.com/brunexgeek/hugye/pkg/cache"
	"github.com/brunexgeek/hugye/pkg/dns"
	"github.com/brunexgeek/hugye/pkg/worker"
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

func preprocess_job(ctx *worker.WorkerContext, job *worker.Job) (*worker.Job, error) {
	response := ctx.Processor.Cache.Get(job.Request.Message.Question[0].Name,
		job.Request.Message.Question[0].Type)
	if response != nil {
		job.Response.Bytes = response
		job.Response.Done = true
		job.Ticket = &dns.Ticket{Id: ctx.Resolver.NextId(), Conn: nil} // TODO: make ticket nil here
		fmt.Println("-- Cache used")
	} else {
		// send query to external DNS
		ticket, error := ctx.Resolver.Send(job.Request.Message.Question[0].Name, job.Request.Bytes, ctx.Resolver.NextId())
		if error != nil {
			return nil, error
		}
		job.Ticket = ticket
	}
	return job, nil
}

func worker_routine(ctx *worker.WorkerContext) error {
	wait_list := make(map[uint16]*worker.Job)

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
			case job := <-ctx.Input:
				job, err := preprocess_job(ctx, job)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if job != nil {
					//fmt.Printf("Job #%d accepted\n", job.Id)
					wait_list[job.Ticket.Id] = job
				}
			case <-time.After(timeout * time.Millisecond):

			}
		}

		//
		// Process each UDP response
		//
		for _, job := range wait_list {
			if job.Response.Done || job.Ticket == nil {
				continue
			}
			//fmt.Println("Checking for responses")
			buffer, err := ctx.Resolver.Receive(job.Ticket, 1)
			if err != nil {
				//fmt.Println(err)
				break
			}

			var id uint16
			_, err = binary.Read16(buffer, 0, &id)
			if err != nil {
				continue
			}
			//fmt.Printf("-- Found response #%d\n", id)

			job.Response.Bytes = buffer
			job.Response.Done = true

			job.Ticket.Conn = nil
			//job.Ticket = nil

			ctx.Processor.Cache.Set(job.Request.Message.Question[0].Name,
				job.Request.Message.Question[0].Type, job.Response.Bytes)
		}

		//
		// Check jobs in the waiting list to finish or discard them.
		//
		// TODO: merge with the above loop?
		for _, job := range wait_list {
			if job.Response.Done {
				// replace the ID
				binary.Write16(job.Response.Bytes, 0, job.Request.Message.Header.Id)
				// send response to client
				size, err := job.Conn.WriteTo(job.Response.Bytes, job.Request.Remote)
				if err != nil {
					fmt.Println(err)
				} else if size != len(job.Response.Bytes) {
					fmt.Println("Unable to send all data")
				}
				delete(wait_list, job.Ticket.Id)
			}
		}
	}
	return fmt.Errorf("Worker done")
}

func main() {
	install_signal_hook()

	config, err := LoadConfig(os.Args[1])
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp4", config.Binding)
	if err != nil {
		panic(err)
	}
	dur := time.Second * 1
	conn.SetDeadline(time.Now().Add(dur))
	buffer := make([]byte, 1024)

	cache := cache.NewCache()
	processor := worker.NewProcessor(cache)
	resolver, err := dns.NewResolver(config.ExternalDNS)
	work, err := processor.StartWorker(resolver, worker_routine)
	if work == nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Worker started")

	for running {
		conn.SetDeadline(time.Now().Add(dur))
		size, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			//fmt.Println(error)
			continue
		}

		job, err := worker.NewJob(conn, remote, buffer, size)
		if err != nil {
			continue
		}
		fmt.Printf("[%d] %6s %s\n", job.Request.Message.Header.Id,
			dns.TypeToString(int(job.Request.Message.Question[0].Type)),
			job.Request.Message.Question[0].Name)
		work.GetContext().Input <- job
		job = nil
	}
	processor.Await()
}
