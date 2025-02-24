# A Lua VM pool in Go

This Lua VM pool is designed to work with my fork of [Shopify's go-lua](https://github.com/epikur-io/go-lua) implementation.


## Example:

```go
package main 

import (
    "log"
    lpool "github.com/epikur-io/go-lua-pool"
)

func main() {
    pool := lpool.NewPool(10)

    // get a VM:
    luaVM := pool.Acquire()

    // do stuff...

    // release VM
    pool.Release(luaVM)

    // get a VM or timeout after 1 second:
    luaVM, err := pool.AcquireTimeout(time.Seconds * 1)
    if (err != nil) {
        log.Println("error:", err)
    }else{
        // do stuff...


        // release VM
        pool.Release(luaVM)
    }
}
```