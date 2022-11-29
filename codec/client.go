package codec

import (
	"bufio"
	"io"
	"net/rpc"
	"sync"
)

type ClientCodec struct {
	rwc     io.ReadWriteCloser
	wBuf    *bufio.Writer
	readMu  sync.Mutex
	writeMu sync.Mutex
}

func NewCBORClientCodec(rwc io.ReadWriteCloser) *ClientCodec {
	return &ClientCodec{
		rwc:  rwc,
		wBuf: bufio.NewWriter(rwc),
	}
}

func (c *ClientCodec) WriteRequest(request *rpc.Request, payload any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return writeAny(c.wBuf, request, payload)
}

func (c *ClientCodec) ReadResponseHeader(response *rpc.Response) error {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	return readAny(c.rwc, response)
}

func (c *ClientCodec) ReadResponseBody(body any) error {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	return readAny(c.rwc, body)
}

func (c *ClientCodec) Close() error {
	return c.rwc.Close()
}
