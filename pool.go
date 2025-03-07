package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	lua "github.com/epikur-io/go-lua"
)

var ErrFailedToReleaseVM = fmt.Errorf("failed to release vm")

// Lua VM pool

type IPool interface {
	Len() int
	Cap() int
	Update()
	UpdateWithTimeout(time.Duration) (int, int)
	Acquire() *lua.State
	AcquireWithTimeout(time.Duration) (*lua.State, error)
	AcquireWithContext(context.Context) (*lua.State, error)
	Release(*lua.State)
	TryRelease(*lua.State) error
	TryReleaseWithContext(context.Context, *lua.State) error
}

// ensure interface is satisfied
var _ IPool = &Pool{}

// Default factory function to create Lua VMs
func NewLuaVM() *lua.State {
	lvm := lua.NewState()
	lua.OpenLibraries(lvm)
	return lvm
}

// Creates a new pool of Lua VMs with the given size/capacity
func NewPool(size int, vmFactoryFunc func() *lua.State) *Pool {
	lp := Pool{size: size, creator: vmFactoryFunc}
	lp.init()
	return &lp
}

type Pool struct {
	// size of the pool
	size int
	// factory function to create Lua VMs
	creator func() *lua.State
	pool    chan *lua.State
	mux     sync.Mutex
}

func (p *Pool) init() {
	p.mux = sync.Mutex{}
	p.pool = make(chan *lua.State, p.size)
	// fill the pool
	for i := 0; i < p.size; i++ {
		p.pool <- p.createVM()
	}
}

func (p *Pool) createVM() *lua.State {
	var lvm *lua.State
	if p.creator != nil {
		lvm = p.creator()
	} else {
		lvm = NewLuaVM()
	}
	return lvm
}

func (p *Pool) Len() int {
	return len(p.pool)
}

func (p *Pool) Cap() int {
	return cap(p.pool)
}

func (p *Pool) Update() {
	// Make sure the pool is empty so we don't miss a vm because
	// it was acquired by an other function
	// So this loop can take a while if some vm's are already acquired and busy.
	p.mux.Lock()
	defer p.mux.Unlock()

	for i := 0; i < cap(p.pool); i++ {
		// empty the Pool
		<-p.pool
	}
	for i := 0; i < cap(p.pool); i++ {
		// fill the Pool
		p.pool <- p.createVM()
	}
}

func (p *Pool) UpdateWithTimeout(to time.Duration) (removedInstanceCount int, newInstanceCount int) {
	p.mux.Lock()
	defer p.mux.Unlock()

	c := time.After(to)
	for i := 0; i < cap(p.pool); i++ {
		// try to empty the Pool
		select {
		case <-p.pool:
			removedInstanceCount++
		case <-c:
			return
		}
	}
	for i := 0; i < cap(p.pool); i++ {
		// try to fill the Pool
		select {
		case p.pool <- p.createVM():
			newInstanceCount++
		case <-c:
			return
		}

	}
	return
}

func (p *Pool) AcquireWithTimeout(to time.Duration) (*lua.State, error) {
	c := time.After(to)
	select {
	case vm := <-p.pool:
		return vm, nil
	case <-c:
		return nil, errors.New("timeout")
	}
}

func (p *Pool) AcquireWithContext(ctx context.Context) (*lua.State, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case vm := <-p.pool:
		return vm, nil
	}
}

// Acquire a vm from the pool (blocking)
func (p *Pool) Acquire() *lua.State {
	return <-p.pool
}

// Releases a vm to the pool (blocking)
// if vm is nil a new vm gets created on the fly
func (p *Pool) Release(vm *lua.State) {
	if vm == nil {
		p.pool <- p.createVM()
		return
	}
	p.pool <- vm
}

// Try to release a vm to the pool (non-blocking)
// if vm is nil a new vm gets created on the fly
func (p *Pool) TryRelease(vm *lua.State) error {
	if vm == nil {
		vm = p.createVM()
	}
	select {
	case p.pool <- vm:
	default:
		return ErrFailedToReleaseVM
	}
	return nil
}

// Try to release a vm to the pool (non-blocking)
// if vm is nil a new vm gets created on the fly
func (p *Pool) TryReleaseWithContext(ctx context.Context, vm *lua.State) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if vm == nil {
		vm = p.createVM()
	}
	select {
	case p.pool <- vm:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
