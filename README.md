= Go net/rpc CBOR codec

This library provides `codec` package with CBOR (binary JSON) serialization for Go's `net/rpc` package. It uses the
reliable and fast CBOR encoding library `github.com/fxamacker/cbor`.

In addition, the `cmd` package provides subprocess IPC mechanism through pipelines

== Example (codec)

The `net/rpc` package is a consistent, stable and easy to use Go RPC mechanism, from the documentation: "The net/rpc
package is frozen and is not accepting new features." To use the CBOR codec, follow
the [net/rpc](https://pkg.go.dev/net/rpc) documentation and use `ClientWithCodec` and `ServeCodec` functions.

Use the `net/rpc` API to perform the calls:

```go
args := &struct{A, B int}{7, 8}
var reply int
client.Call("Arith.Multiply", args, &reply)
fmt.Printf("Arith: %d*%d=%d", args.A, args.B, reply)
```

=== The protocol

The protocol is binary and very simple, each data is prefixed with size (uint32, little-endian)

Request:

| Size (bytes) | Type        | Description              |
|--------------|-------------|--------------------------|
| 4            | uint32 (LE) | Size of the next block   |
| ?            | CBOR data   | RPC header data frame    |
| 4            | uint32 (LE) | Size of the next block   |
| ?            | CBOR data   | RPC arguments data frame |

Response:

| Size (bytes) | Type        | Description              |
|--------------|-------------|--------------------------|
| 4            | uint32 (LE) | Size of the next block   |
| ?            | CBOR data   | RPC reply data frame     |

Example data frame values:

* Request header data: `{'Seq': 1, 'ServiceMethod': 'Arith.Multiply'}}`
* Request arguments data: `{'A': 7, 'B': 8}`
* Response header data: `{'Seq': 1, 'ServiceMethod': 'Arith.Multiply', 'Error': ''}}`
* Response reply data: `56`

Errors are string values.

== Example (Python interoperability)

Here is a quick and dirty example on how to use this library to spawn a Python subprocess and communicate with it over
pipes. For a complete example, see [internal/examples/python](internal/examples/python) directory.

It is based on the `net/rpc` package from Go:

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
	proc, _ := cmd.NewCommand(context.TODO(), "python3", "internal/examples/python/service.py")
	proc.Start(context.TODO())
	defer proc.Stop(context.TODO())

	args := &Args{7, 8}
	var reply int

	// Synchronous call
	proc.Call("Arith.Multiply", args, &reply)
	fmt.Printf("Multiply (sync): %d*%d=%d\n", args.A, args.B, reply)

	// Asynchronous call
	call := proc.Go("Arith.Multiply", args, &reply, nil)
	<-call.Done
	fmt.Printf("Multiply (assync): %d*%d=%d\n", args.A, args.B, reply)
}
```

To implement a subprocess in any language, here is the contract:

* The protocol is binary, make sure to read standard input and write to standard output accordingly.
* Flush the write buffer after each response.
* Check the input for EOF and terminate the program when it's reached.
* The standard error is being written into the client log, use it for logging.
* Make sure to synchronize IO when doing concurrent handling (threads).

For an example, open [internal/examples/python/server.py](internal/examples/python/server.py).