package codec_test

import (
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"sync"
	"testing"

	"github.com/lzap/cborpc/codec"
)

type pipePair struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (pp *pipePair) Read(p []byte) (int, error) {
	return pp.reader.Read(p)
}

func (pp *pipePair) Write(p []byte) (int, error) {
	return pp.writer.Write(p)
}

func (pp *pipePair) Close() error {
	pp.writer.Close()
	return pp.reader.Close()
}

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

type Arith struct{}

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

func server(codec rpc.ServerCodec) {
	err := rpc.Register(new(Arith))
	if err != nil {
		panic(err)
	}
	rpc.ServeCodec(codec)
}

var serverOnce = sync.Once{}

var client *rpc.Client

func setup() {
	var in, out pipePair
	in.reader, out.writer = io.Pipe()
	out.reader, in.writer = io.Pipe()

	go server(codec.NewCBORServerCodec(&out))
	client = rpc.NewClientWithCodec(codec.NewCBORClientCodec(&in))
}

func TestMultiplySync(t *testing.T) {
	serverOnce.Do(setup)

	args := &Args{7, 8}
	var reply int
	err := client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		t.Fatal(err)
	}
	if reply != 56 {
		t.Fatalf("result is incorrect: %d", reply)
	}
}

func TestDivideAsync(t *testing.T) {
	serverOnce.Do(setup)

	args := &Args{7, 8}
	quotient := new(Quotient)
	divCall := client.Go("Arith.Divide", args, quotient, nil)
	<-divCall.Done
	fmt.Printf("Divide (async): %d/%d=%+v\n", args.A, args.B, quotient)

	if divCall.Error != nil {
		t.Fatal(divCall.Error)
	}
	if quotient.Quo != 0 && quotient.Rem != 7 {
		t.Fatalf("result is incorrect: %+v", quotient)
	}
}
