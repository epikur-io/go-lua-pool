package pool

import (
	"errors"
	"time"

	lua "github.com/epikur-io/go-lua"
)

func NewPool(size int) *Pool {
	lp := Pool{Size: size}
	lp.init()
	return &lp
}

type Pool struct {
	// Size of the pool
	Size int
	// Custructor function to create Lua VMs
	Creator func() *lua.State
	pool    chan *lua.State
}

func (p *Pool) init() {
	p.pool = make(chan *lua.State, p.Size)
	// fill the pool
	for i := 0; i < p.Size; i++ {
		var lvm *lua.State
		if p.Creator != nil {
			lvm = p.Creator()
		} else {
			lvm = NewLuaVM()
		}
		p.pool <- lvm
	}
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
		var lvm *lua.State
		if p.Creator != nil {
			lvm = p.Creator()
		} else {
			lvm = NewLuaVM()
		}
		p.pool <- lvm
	}
}

func (p *Pool) AcquireTimeout(to time.Duration) (*lua.State, error) {
	c := time.After(to)
	select {
	case vm := <-p.pool:
		return vm, nil
	case <-c:
		return nil, errors.New("timeout")
	}
}

func (p *Pool) Acquire() *lua.State {
	return <-p.pool
}

func (p *Pool) Release(vm *lua.State) {
	if vm == nil {
		var lvm *lua.State
		if p.Creator != nil {
			lvm = p.Creator()
		} else {
			lvm = NewLuaVM()
		}
		p.pool <- lvm
		return
	}
	p.pool <- vm
}

// Default constructor to create Lua VMs
func NewLuaVM() *lua.State {
	lvm := lua.NewState()
	lua.OpenLibraries(lvm)
	return lvm
}
