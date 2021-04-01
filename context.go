package contextimpl

import (
	"errors"
	"reflect"
	"sync"
	"time"
)

type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}

type emtyContext int

func (emtyContext) Deadline() (deadline time.Time, ok bool) { return }
func (emtyContext) Done() <-chan struct{}                   { return nil }
func (emtyContext) Err() error                              { return nil }
func (emtyContext) Value(key interface{}) interface{}       { return nil }

var (
	background = new(emtyContext)
	todo       = new(emtyContext)
)

func Background() Context { return background }

func TODO() Context { return todo }

type cancelCtx struct {
	Context
	done chan struct{}
	err  error
	mu   sync.Mutex
}

func (ctx *cancelCtx) Done() <-chan struct{} { return ctx.done }
func (ctx *cancelCtx) Err() error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.err
}

func (ctx *cancelCtx) cancel(err error) {
	ctx.mu.Lock()
	if ctx.err != nil {
		return
	}
	ctx.err = err
	close(ctx.done)
	ctx.mu.Unlock()
}

type CancelFunc func()

var Canceled = errors.New("context cancelled")

func WithCancel(parent Context) (Context, CancelFunc) {
	ctx := &cancelCtx{
		Context: parent,
		done:    make(chan struct{}),
	}
	cancel := func() {
		ctx.cancel(Canceled)
	}

	go func() {
		select {
		case <-parent.Done():
			ctx.cancel(parent.Err())
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

type DeadlineCtx struct {
	*cancelCtx
	deadline time.Time
}

func (ctx *DeadlineCtx) Deadline() (deadline time.Time, ok bool) {
	return ctx.deadline, true
}

var DeadlineExceeded = errors.New("deadline exceeded")

func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	cctx, cancel := WithCancel(parent)

	ctx := &DeadlineCtx{
		cancelCtx: cctx.(*cancelCtx),
		deadline:  deadline,
	}
	t := time.AfterFunc(time.Until(deadline), func() { ctx.cancel(DeadlineExceeded) })

	stop := func() {
		t.Stop()
		cancel()
	}

	return ctx, stop
}

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

type valueContext struct {
	Context
	key, value interface{}
}

func (ctx *valueContext) Value(key interface{}) interface{} {
	if ctx.key == key {
		return ctx.value
	}
	return ctx.Context.Value(key)
}

func WithValue(parent Context, key, value interface{}) Context {
	if key == nil {
		panic("key is nil")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueContext{
		Context: parent,
		key:     key,
		value:   value,
	}
}
