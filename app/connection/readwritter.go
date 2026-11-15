package connection

import (
	"bufio"
	"io"
	"net"
	"sync"
)

type ConnRW struct {
	rw  *bufio.ReadWriter
	rmu sync.Mutex
	wmu sync.Mutex
}

func NewConnRW(conn net.Conn) ConnRW {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	rw := bufio.NewReadWriter(r, w)
	return ConnRW{rw: rw}
}

func (crw *ConnRW) Write(data []byte) (int, error) {
	crw.wmu.Lock()
	defer crw.wmu.Unlock()
	var n int
	n, err := crw.rw.Write(data)
	if err != nil {
		return n, err
	}
	err = crw.rw.Flush()
	if err != nil {
		return n, err
	}
	return n, nil
}

func (crw *ConnRW) Read(buf []byte) (int, error) {
	crw.rmu.Lock()
	defer crw.rmu.Unlock()
	return crw.rw.Reader.Read(buf)
}

func (crw *ConnRW) ReadBytes(delim byte) ([]byte, error) {
	crw.rmu.Lock()
	defer crw.rmu.Unlock()
	return crw.rw.ReadBytes(delim)
}

func (crw *ConnRW) ReadFull(buf []byte) (int, error) {
	crw.rmu.Lock()
	defer crw.rmu.Unlock()
	return io.ReadFull(crw.rw.Reader, buf)
}
