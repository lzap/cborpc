package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/rpc"
	"os/exec"
    "sync"
    "syscall"
	"time"

	"github.com/lzap/cborpc/codec"
	"github.com/lzap/cborpc/log"
)

type readWriteCloser struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (rwc *readWriteCloser) Read(p []byte) (int, error) {
	return rwc.reader.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (int, error) {
	return rwc.writer.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	err := rwc.writer.Close()
	err = rwc.reader.Close()
	return err
}

type Command struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stdin  io.WriteCloser
	stderr io.ReadCloser
	client *rpc.Client
    inflightWG sync.WaitGroup
}

func NewCommand(ctx context.Context, name string, arg ...string) (*Command, error) {
	cmd := exec.Command(name, arg...)
	logger := log.ContextLogger(ctx)
	logger.Msgf(log.TRC, "Executing command %s with arguments %v", name, arg)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe error for '%s' PID %d: %w", cmd.Path, cmd.Process.Pid, err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe error for '%s' PID %d: %w", cmd.Path, cmd.Process.Pid, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe error for '%s' PID %d: %w", cmd.Path, cmd.Process.Pid, err)
	}

	command := Command{
		cmd:    cmd,
		stdout: stdout,
		stdin:  stdin,
		stderr: stderr,
	}
	return &command, nil
}

func (cmd *Command) Start(ctx context.Context) error {
	err := cmd.cmd.Start()

	if err != nil {
		return fmt.Errorf("cannot start command '%s': %w", cmd.cmd.Path, err)
	}

	out := readWriteCloser{
		reader: cmd.stdout,
		writer: cmd.stdin,
	}
	cmd.client = rpc.NewClientWithCodec(codec.NewCBORClientCodec(&out))

	go cmd.readerStderr(ctx)

	return nil
}

func (cmd *Command) readerStderr(ctx context.Context) {
	logger := log.ContextLogger(ctx)
	logger.Msgf(log.TRC, "Starting stderr reader")
	defer logger.Msgf(log.TRC, "Stopping stderr reader")

	scanner := bufio.NewScanner(cmd.stderr)
	for scanner.Scan() {
		logger.Msgf(log.WRN, "[%d] %s", cmd.cmd.Process.Pid, scanner.Text())
	}
}

// Stop for all "in flight" calls to be finished first, then it closes the pipe
// and sends termination signal to the subprocess and waits until the process exits.
func (cmd *Command) Stop(ctx context.Context) error {
    cmd.inflightWG.Wait()

    logger := log.ContextLogger(ctx)
	logger.Msgf(log.TRC, "Stopping RPC client, terminating process, waiting for data")
	defer logger.Msgf(log.TRC, "All data processed")

	err := cmd.client.Close()
	if err != nil {
		return err
	}

	err = cmd.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("terminate error for process %d: %w", cmd.cmd.Process.Pid, err)
	}

	err = cmd.cmd.Wait()
	return err
}

// Call performs RPC call, the context is not propagated to the server. It should be used for
// client-only timeout. Since contexts are not passed to the server, the call will continue
// execution on the server even if the client was already cancelled and possibly can block
// other subsequent calls.
func (cmd *Command) Call(ctx context.Context, serviceMethod string, args any, reply any) error {
    cmd.inflightWG.Add(1)
	logger := log.ContextLogger(ctx)
	start := time.Now()
	logger.Msgf(log.TRC, "Call: %s started", serviceMethod)
    defer func() {
        logger.Msgf(log.TRC, "Call: %s finished, duration: %s", serviceMethod, time.Since(start))
        cmd.inflightWG.Done()
    }()

	call := cmd.client.Go(serviceMethod, args, reply, make(chan *rpc.Call, 1))
	select {
	case <-call.Done:
		return call.Error
	case <-ctx.Done():
		return ctx.Err()
	}
}
