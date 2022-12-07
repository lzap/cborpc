# Go net/rpc CBOR codec

This library provides `codec` package with CBOR (binary JSON) serialization for Go's `net/rpc` package. It uses the
reliable and fast CBOR encoding library `github.com/fxamacker/cbor`.

In addition, the `cmd` package provides subprocess RPC-like IPC mechanism through pipelines for easy Go to _any
language_ integration. Since Go `net/rpc` is extremely easy to use, it is easy to create a service/server/plugin in
any language. A Python example is available. The CBOR codec is used for the IPC protocol because the default Go
RPC [Gob serialization](https://pkg.go.dev/encoding/gob) is not widely available on other languages and runtimes.

## Example (codec)

The `net/rpc` package is a consistent, stable and easy to use Go RPC mechanism. To use the CBOR codec, follow
the [net/rpc](https://pkg.go.dev/net/rpc) documentation and use `ClientWithCodec` and `ServeCodec` functions.

To perform a call:

```go
args := &struct{A, B int}{7, 8}
var reply int

// in := ReadWriteCloser (pipe, socket, HTTP connection...)
client = rpc.NewClientWithCodec(codec.NewCBORClientCodec(&in))
client.Call("Arith.Multiply", args, &reply)

fmt.Printf("Arith: %d*%d=%d", args.A, args.B, reply)
```

To implement a service:

```go
type Arith struct{}

func (t *Arith) Multiply(args *Args, reply *int) error {
    *reply = args.A * args.B
    return nil
}

func server(codec rpc.ServerCodec) {
    err := rpc.Register(new(Arith))
    rpc.ServeCodec(codec)
}

// out := ReadWriteCloser (pipe, socket, HTTP connection...)
go server(codec.NewCBORServerCodec(&out))
```

### The protocol

The protocol is binary, each data frame is prefixed with size (uint32, little-endian).

**Request**:

| Size (bytes) | Type        | Description              |
|--------------|-------------|--------------------------|
| 4            | uint32 (LE) | Size of the next block   |
| ?            | CBOR data   | RPC header data frame    |
| 4            | uint32 (LE) | Size of the next block   |
| ?            | CBOR data   | RPC arguments data frame |

**Response**:

| Size (bytes) | Type        | Description            |
|--------------|-------------|------------------------|
| 4            | uint32 (LE) | Size of the next block |
| ?            | CBOR data   | RPC reply data frame   |
| 4            | uint32 (LE) | Size of the next block |
| ?            | CBOR data   | RPC reply data frame   |

Example data frame values:

* Request header data: `{'Seq': 1, 'ServiceMethod': 'Arith.Multiply'}}`
* Request arguments data: `{'A': 7, 'B': 8}`
* Response header data: `{'Seq': 1, 'ServiceMethod': 'Arith.Multiply', 'Error': ''}}`
* Response reply data: `56`

Errors are string values, just like in Go. An empty string indicates no error.

## Example (Python interoperability)

Here is a quick and dirty example on how to use this library to spawn a Python subprocess and perform GPC-like
communication over pipes. Only Go -> Python direction is currently implemented at the moment as this was the reason for
my use case, contributions are welcome.

For a complete example, see [internal/examples/python](internal/examples/go-calls-python) directory. An example
calling **Python subprocess** from Go:

```go
package main

import (
	"context"
	"fmt"

	"github.com/lzap/cborpc/cmd"
)

type Args struct {
	A, B int
}

func main() {
	ctx := context.TODO()
	proc, _ := cmd.NewCommand(ctx, "python3", "internal/examples/go-calls-python/service.py")
	proc.Start()
	defer proc.Stop(ctx)

	args := &Args{7, 8}
	var reply int

	// Synchronous call
	proc.Call(ctx, "Arith.Multiply", args, &reply)

	fmt.Printf("Multiply (sync): %d*%d=%d\n", args.A, args.B, reply)

	// Asynchronous call
	call := proc.Go(ctx, "Arith.Multiply", args, &reply, nil)
	<-call.Done

	fmt.Printf("Multiply (assync): %d*%d=%d\n", args.A, args.B, reply)
}
```

To implement a subprocess in any language, here is the contract:

* The protocol is binary, do not read standard input and write to standard output in text mode.
* Flush the write buffer after each response otherwise the communication will get stuck.
* Check the input for EOF and terminate the program when it's reached.
* The standard error is being written into the client log, use it for logging.
* Make sure to synchronize IO when doing concurrent handling (threads).
* Writing single-thread services is a valid approach, make sure to scale out via multiple processes.

For an example of Python subprocess,
see [internal/examples/python/server.py](internal/examples/go-calls-python/server.py).

## Performance

This library is not tuned or tested for the best performance, but generally CBOR over pipes should be reasonably fast.
Expect faster performance than TCP sockets, almost the same performance as UNIX sockets and of course slower performance
than shared memory or UNIX message queue.

This code is ideal when you need to call a script few calls per minute avoiding spawning process for each call. All
communication is synchronized and safe to use from multiple goroutines, the Python example is single thread so calls
will block. Spawn multiple commands to scale amount of calls up.

## Contributing

This package is frozen and is not accepting new features, however, bug fixes or examples for other languages are
welcome.

## LICENSE

MIT