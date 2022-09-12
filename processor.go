package main

import (
	"context"
	"fmt"
	"net"
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
	id         uint16 // external DNS message id (zero means empty)
	max        int    // number of external DNS queries made
	count      int    // number of external DNS responses received
	start_time int64  // time which the processing started
}

type Processor struct {
	workers []WorkerContext
}

type WorkerContext struct {
	processor *Processor
	input     chan Job
	output    chan Job
}

type Future interface {
	Await() error
	GetContext() *WorkerContext
}

type future struct {
	await       func(ctx context.Context) error
	get_context func() *WorkerContext
}

func (f future) Await() error {
	return f.await(context.Background())
}

func (f future) GetContext() *WorkerContext {
	return f.get_context()
}

// Exec executes the async function
func (p *Processor) StartWorker(f func(*WorkerContext) error) Future {
	var result error
	ctx := &WorkerContext{processor: p, input: make(chan Job, 5)}
	fmt.Println(ctx)
	c := make(chan struct{})
	go func() {
		defer close(c)
		result = f(ctx)
	}()
	return future{
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
}
