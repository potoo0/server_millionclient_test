package public

import (
	"crypto/tls"
	"net"
	"reflect"
	"sync"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

type Epoll struct {
	fd          int
	connections map[int]net.Conn
	lock        *sync.RWMutex
}

func MkEpoll() (*Epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &Epoll{
		fd:          fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]net.Conn),
	}, nil
}

func (e *Epoll) Add(conn net.Conn) error {
	fd := netFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.connections[fd] = conn
	if len(e.connections)%100 == 0 {
		Logger.Debug("", zap.Int("connections", len(e.connections)))
	}
	return nil
}

func (e *Epoll) Remove(conn net.Conn) error {
	fd := netFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.connections, fd)
	if len(e.connections)%100 == 0 {
		Logger.Debug("", zap.Int("connections", len(e.connections)))
	}
	return nil
}

func (e *Epoll) Wait() ([]net.Conn, error) {
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(e.fd, events, -1)
	if err != nil {
		return nil, err
	}
	e.lock.RLock()
	defer e.lock.RUnlock()
	var connections []net.Conn
	for i := 0; i < n; i++ {
		conn := e.connections[int(events[i].Fd)]
		connections = append(connections, conn)
	}
	return connections, nil
}

func netFD(conn net.Conn) int {
	//connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	//tcpConn := reflect.Indirect(connVal).FieldByName("conn")'
	var tcpConn reflect.Value
	if _, ok := conn.(*tls.Conn); ok {
		connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
		tcpConn = reflect.Indirect(connVal).FieldByName("conn")
	} else {
		tcpConn = reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	}
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}
