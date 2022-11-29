package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/rpc"
	"os/exec"
	"syscall"

	"github.com/lzap/cborpc/codec"
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
	rwc.writer.Close()
	return rwc.reader.Close()
}

type Command struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stdin  io.WriteCloser
	stderr io.ReadCloser
	client *rpc.Client
}

func NewCommand(ctx context.Context, name string, arg ...string) (*Command, error) {
	cmd := exec.Command(name, arg...)

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
	scanner := bufio.NewScanner(cmd.stderr)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

func (cmd *Command) Stop(ctx context.Context) error {
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

func (cmd *Command) Call(serviceMethod string, args any, reply any) error {
	return cmd.client.Call(serviceMethod, args, reply)
}

func (cmd *Command) Go(serviceMethod string, args any, reply any, done chan *rpc.Call) *rpc.Call {
	return cmd.client.Go(serviceMethod, args, reply, done)
}
