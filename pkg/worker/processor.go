package worker

import (
	"container/list"
	"context"
	"net"
	"time"

	"github.com/brunexgeek/hugye/pkg/dns"
	"github.com/brunexgeek/hugye/pkg/domain"
)

type Job struct {
	Conn    *net.UDPConn
	Request struct {
		Message *dns.Message // parsed from 'Bytes'
		Bytes   []byte       // DNS message from client
		Remote  *net.UDPAddr // client address
	}
	Response struct {
		Bytes []byte // DNS message to client
		Done  bool   // ready to send the response to the client?
	}
	external struct {
		max   int // number of external DNS queries made
		count int // number of external DNS responses received
	}
	Id        uint16 // external DNS message id (zero means empty)
	StartTime int64  // time which the processing started
}

type Processor struct {
	workers *list.List
	Cache   domain.Cache
}

type WorkerContext struct {
	Processor *Processor
	Input     chan *Job
	output    chan *Job
	Resolver  domain.Resolver // connection with external DNS
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
func (p *Processor) StartWorker(resolver domain.Resolver, fun func(*WorkerContext) error) (Worker, error) {
	var result error
	ctx := &WorkerContext{
		Processor: p,
		Input:     make(chan *Job, 20),
		Resolver:  resolver,
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		result = fun(ctx)
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
	p.workers.PushBack(&worker)
	return worker, nil
}

func (p *Processor) Await() {
	for it := p.workers.Front(); it != nil; it = it.Next() {
		it.Value.(*internal_worker).Await()
	}
}

func NewJob(conn *net.UDPConn, addr *net.UDPAddr, buf []byte, size int) (*Job, error) {
	var err error

	job := Job{}
	job.Conn = conn
	job.StartTime = time.Now().UnixMilli()
	job.Request.Remote = addr
	job.Request.Message, err = dns.ParseMessage(buf)
	if err != nil {
		return nil, err
	}

	if len(buf) != size {
		job.Request.Bytes = make([]byte, size)
		copy(job.Request.Bytes, buf[:size])
	} else {
		job.Request.Bytes = buf
	}

	return &job, nil
}

func NewProcessor(cache domain.Cache) *Processor {
	result := Processor{workers: list.New(), Cache: cache}
	return &result
}
