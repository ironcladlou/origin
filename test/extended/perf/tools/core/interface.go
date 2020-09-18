package core

import (
	"context"
	"sync"
	"time"
)

type TestContext struct {
	TestCancel context.Context
	WaitGroup  *sync.WaitGroup
}

type WorkerContext struct {
	Name string
	Shutdown context.Context
	WaitGroup *sync.WaitGroup
}

// represents a worker.
type Worker func(*WorkerContext)
func (w Worker) Work(context *WorkerContext) {
	w(context)
}

// Runner runs a worker
type Runner func(*WorkerContext, Worker)
func (r Runner) Run(wc *WorkerContext, worker Worker) {
	r(wc, worker)
}

type Action func()
func (a Action) Do() {
	a()
}


// LoadGenerator tunes a set of Doer.
type LoadGenerator func([]Action)
func (l LoadGenerator) Generate(actions []Action) {
	l(actions)
}


func NewTestContext(parent context.Context, duration time.Duration) (*TestContext, context.CancelFunc) {
	wg := &sync.WaitGroup{}
	c, cancel := context.WithTimeout(parent, duration)

	return &TestContext{
		TestCancel: c,
		WaitGroup:  wg,
	}, cancel
}