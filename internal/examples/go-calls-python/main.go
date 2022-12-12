package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
    "time"

    "github.com/lzap/cborpc/cmd"
	"github.com/lzap/cborpc/log"
)

type Args struct {
	A, B int
}

func main() {
	ctx := log.ContextWithStdoutLogger(context.Background())
	logger := log.ContextLogger(ctx)

	script := "server.py"
	if len(os.Args) > 1 {
		script = os.Args[1]
	}

	proc, err := cmd.NewCommand(ctx, "python3", script)
	if err != nil {
		panic(err)
	}
	err = proc.Start(ctx)
	if err != nil {
		panic(err)
	}

	args := &Args{7, 8}
	var reply int

    ctx, cancel := context.WithTimeout(ctx, time.Second)
    defer cancel()
	err = proc.Call(ctx, "Arith.Multiply", args, &reply)
	if err != nil {
		logger.Msgf(log.ERR, "call error: %w", err)
	}
	fmt.Printf("Multiply: %d*%d=%d\n", args.A, args.B, reply)

	defer func() {
		err := proc.Stop(ctx)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fmt.Printf("process exited with code: %d\n", exitErr.ProcessState.ExitCode())
		} else if err != nil {
			panic(err)
		}
	}()
}
