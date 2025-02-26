package pool

import (
	"errors"
	"fmt"
	"time"

	lua "github.com/epikur-io/go-lua"
)

var ErrFailedToReleaseVM = fmt.Errorf("failed to release vm")

// Creates a new pool of Lua VMs with the given size/capacity
func NewPool(size int) *Pool {
	lp := Pool{size: size}
	lp.init()
	return &lp
}

type Pool struct {
	// size of the pool
	size int
	// factory function to create Lua VMs
	Creator func() *lua.State
	pool    chan *lua.State
}

func (p *Pool) init() {
	p.pool = make(chan *lua.State, p.size)
	// fill the pool
	for i := 0; i < p.size; i++ {
		p.pool <- p.createVM()
	}
}

func (p *Pool) createVM() *lua.State {
	var lvm *lua.State
	if p.Creator != nil {
		lvm = p.Creator()
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
		select {
		case p.pool <- p.createVM():
		default:
			return ErrFailedToReleaseVM
		}
		return nil
	}
	select {
	case p.pool <- vm:
	default:
		return ErrFailedToReleaseVM
	}
	return nil
}

// Default factory function to create Lua VMs
func NewLuaVM() *lua.State {
	lvm := lua.NewState()
	lua.OpenLibraries(lvm)
	return lvm
}
