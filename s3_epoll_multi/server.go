package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	_ "net/http/pprof"
	"runtime"
	"server_millionclient/public"
	"server_millionclient/public/protocol"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	addr    = flag.String("addr", ":8000", "server addr")
	verbose = flag.Bool("verbose", false, "verbose")
)

func main() {
	flag.Parse()
	public.InitLogger(*verbose)
	public.SetLimit()

	listenerNum := runtime.NumCPU()
	public.Logger.Info("listening", zap.String("addr", *addr), zap.Int("listenerNum", listenerNum))

	ctx, cancel := context.WithCancelCause(context.TODO())
	var errOnce sync.Once
	for range listenerNum {
		go func() {
			if err := listen(ctx); err != nil {
				errOnce.Do(func() { cancel(err) })
			}
		}()
	}
	if err := context.Cause(ctx); err != nil {
		public.Logger.Error("listen failed", zap.Error(err))
	}

	go public.ServeMetrics()

	<-ctx.Done()
}

func listen(ctx context.Context) error {
	ln, err := public.ListenConfig.Listen(ctx, "tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	// Start epoll
	epoller, err := public.MkEpoll()
	if err != nil {
		return fmt.Errorf("mk epoll failed: %w", err)
	}
	go Start(epoller)

	for {
		conn, err := ln.Accept()
		if err != nil {
			public.Logger.Error("tcp accept failed", zap.Error(err))

			if errors.Is(err.(*net.OpError).Err, net.ErrClosed) {
				return err
			}
			continue
		}

		if err := epoller.Add(conn); err != nil {
			public.Logger.Error("epoller add connection failed", zap.Error(err))
			_ = conn.Close()
		} else {
			public.ConnectionCount.Inc()
		}
	}
}

func Start(epoller *public.Epoll) {
	for {
		connections, err := epoller.Wait()
		// 忽略 EINTR(interrupted system call)
		if errors.Is(err, syscall.EINTR) || len(connections) == 0 {
			runtime.Gosched()
			continue
		}
		if err != nil {
			public.Logger.Error("epoller wait failed", zap.Error(err))
			continue
		}
		for _, conn := range connections {
			if conn == nil {
				break
			}
			if err := handleConn(conn); err != nil {
				public.ConnectionCount.Dec()
				if err := epoller.Remove(conn); err != nil {
					public.Logger.Error("epoller remove connection failed", zap.Error(err))
				}
				_ = conn.Close()
			}
		}
	}
}

func handleConn(conn net.Conn) error {
	defer public.RequestCount.Inc()
	_, body, err := protocol.Read(conn)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	_ = body
	public.Logger.Debug("read", zap.ByteString("body", body))
	var msg public.Msg
	if err = json.Unmarshal(body, &msg); err != nil {
		public.Logger.Info("unmarshal failed", zap.Error(err))
	} else {
		public.Latency.Observe(float64(time.Now().UnixMilli() - msg.Ts))
	}

	// echo
	if bytes, err := protocol.Pack(body); err != nil {
		return fmt.Errorf("pack failed: %w", err)
	} else {
		if _, err := conn.Write(bytes); err != nil {
			return fmt.Errorf("write failed: %w", err)
		}
	}
	return nil
}
