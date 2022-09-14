package main

import (
	"container/list"
	"context"
	"fmt"
	"net"
	"time"
)

type Job struct {
	conn    *net.UDPConn
	request struct {
		bytes  []byte // DNS message from client
		oid    uint16 // original DNS message id (from the client)
		domain string
		remote *net.UDPAddr // client address
	}
	response struct {
		bytes []byte // DNS message to client
	}
	external struct {
		max   int // number of external DNS queries made
		count int // number of external DNS responses received
	}
	id         uint16 // external DNS message id (zero means empty)
	start_time int64  // time which the processing started
}

type Processor struct {
	workers *list.List
	extdns  *net.UDPAddr
}

type WorkerContext struct {
	processor *Processor
	input     chan *Job
	output    chan *Job
	resolver  *Resolver // connection with external DNS
}

type Worker interface {
	Await() error
	GetContext() *WorkerContext
}

type internal_worker struct {
	await       func(ctx context.Context) error
	get_context func() *WorkerContext
}

func (f internal_worker) Await() error {
	return f.await(context.Background())
}

func (f internal_worker) GetContext() *WorkerContext {
	return f.get_context()
}

// Exec executes the async function
func (p *Processor) StartWorker() (Worker, error) {
	resolver, err := MakeResolver(p.extdns)
	if err != nil {
		return nil, err
	}
	var result error
	ctx := &WorkerContext{
		processor: p,
		input:     make(chan *Job, 20),
		resolver:  resolver,
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		result = worker_routine(ctx)
	}()
	worker := internal_worker{
		await: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c:
				return result
			}
		},
		get_context: func() *WorkerContext {
			return ctx
		},
	}
	if p.workers == nil {
		p.workers = list.New()
	}
	p.workers.PushBack(&worker)
	return worker, nil
}

func (p *Processor) Await() {
	for it := p.workers.Front(); it != nil; it = it.Next() {
		it.Value.(*internal_worker).Await()
	}
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

func worker_routine(ctx *WorkerContext) error {
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
			fmt.Printf("-- Found response #%d\n", id)

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
