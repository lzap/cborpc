package codec

import (
	"bufio"
	"io"
	"net/rpc"
	"sync"
)

type ServerCodec struct {
	rwc     io.ReadWriteCloser
	wBuf    *bufio.Writer
	readMu  sync.Mutex
	writeMu sync.Mutex
}

func NewCBORServerCodec(rwc io.ReadWriteCloser) *ServerCodec {
	return &ServerCodec{
		rwc:  rwc,
		wBuf: bufio.NewWriter(rwc),
	}
}

func (c *ServerCodec) ReadRequestHeader(request *rpc.Request) error {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	return readAny(c.rwc, request)
}

func (c *ServerCodec) ReadRequestBody(body any) error {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	return readAny(c.rwc, body)
}

func (c *ServerCodec) WriteResponse(response *rpc.Response, payload any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return writeAny(c.wBuf, response, payload)
}

func (c *ServerCodec) Close() error {
	return c.rwc.Close()
}
