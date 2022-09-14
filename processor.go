package main

import (
	"container/list"
	"context"
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
func (p *Processor) StartWorker(f func(*WorkerContext) error) (Worker, error) {
	resolver, err := MakeResolver(p.extdns)
	if err != nil {
		return nil, err
	}
	var result error
	ctx := &WorkerContext{
		processor: p,
		input:     make(chan *Job, 5),
		resolver:  resolver,
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		result = f(ctx)
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
