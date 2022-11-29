package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"

	"github.com/lzap/cborpc/cmd"
)

type Args struct {
	A, B int
}

func main() {
	proc, err := cmd.NewCommand(context.TODO(), "python3", "server.py")
	if err != nil {
		panic(err)
	}
	err = proc.Start(context.TODO())
	if err != nil {
		panic(err)
	}

	args := &Args{7, 8}
	var reply int

	// Synchronous call
	err = proc.Call("Arith.Multiply", args, &reply)
	if err != nil {
		log.Fatal("arith error: ", err)
	}
	fmt.Printf("Multiply (sync): %d*%d=%d\n", args.A, args.B, reply)

	// Asynchronous call
	call := proc.Go("Arith.Multiply", args, &reply, nil)
	<-call.Done
	if call.Error != nil {
		log.Fatal("arith error: ", call.Error)
	}
	fmt.Printf("Multiply (assync): %d*%d=%d\n", args.A, args.B, reply)

	defer func() {
		err := proc.Stop(context.TODO())
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fmt.Printf("process exited with code: %d\n", exitErr.ProcessState.ExitCode())
		} else if err != nil {
			panic(err)
		}
	}()
}
