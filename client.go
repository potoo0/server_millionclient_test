package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"server_millionclient/public"
	"server_millionclient/public/protocol"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	addr        = flag.String("addr", "127.0.0.1:8000", "server addr")
	connections = flag.Int("conn", 1, "number of websocket connections")
	timeout     = flag.Duration("timeout", 10*time.Second, "timeout for connection")
	goroutines  = flag.Int("goroutines", 100, "number for goroutines")
)

func main() {
	flag.Parse()
	public.InitLogger(true)
	public.SetLimit()

	*connections = max(*connections, 1)
	*goroutines = max(*goroutines, 1)

	public.Logger.Info("Connecting to server",
		zap.String("addr", *addr),
		zap.Int("connections", *connections),
		zap.Duration("timeout", *timeout),
		zap.Int("goroutines", *goroutines),
	)

	// 1. init conns
	conns := make([]net.Conn, *connections)
	err := parallelProcess(conns, *goroutines, func(ctx context.Context, start, end int, conns []net.Conn) error {
		for i := start; i < end; i++ {
			dialer := net.Dialer{Timeout: *timeout}
			conn, err := dialer.DialContext(ctx, "tcp", *addr)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				public.Logger.Error("Failed to connect", zap.Int("idx", i), zap.Error(err))
				return err
			}
			conns[i] = conn
		}
		return nil
	})
	if err != nil {
		return
	}
	if len(conns) == 0 {
		public.Logger.Fatal("No connections established")
	} else {
		public.Logger.Info("Finished initializing connections", zap.Int("cnt", len(conns)))
	}
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	// 2. send bytes
	tts := time.Second
	if *connections > 100 {
		tts = time.Millisecond * 5
	}
	_ = parallelProcess(conns, *goroutines, func(ctx context.Context, start, end int, conns []net.Conn) error {
		timer := time.NewTimer(tts)

		for {
			var diffArr []int64
			for i := start; i < end; i++ {
				conn := conns[i]
				if conn == nil {
					continue
				}

				timer.Reset(tts)
				select {
				case <-ctx.Done():
					return nil
				case <-timer.C:
				}
				ts := time.Now().UnixMilli()
				bytes, err := buildMsg(i, ts)
				if err != nil {
					public.Logger.Error("genMsg failed", zap.Int("idx", i), zap.Error(err))
					return err
				}
				if _, err = conn.Write(bytes); err != nil {
					public.Logger.Error("conn.Write failed", zap.Int("idx", i), zap.Error(err))
					return err
				}
				if reply, err := readMsg(conn); err != nil {
					public.Logger.Error("readMsg failed", zap.Int("idx", i), zap.Error(err))
					return err
				} else {
					diffArr = append(diffArr, time.Now().UnixMilli()-reply.Ts)
				}
			}
			public.Logger.Info("Finished sending messages", zap.Int("cnt", len(diffArr)), zap.Int64("diffAvg", avg(diffArr)))
		}
	})
}

func buildMsg(i int, ts int64) ([]byte, error) {
	msg := public.Msg{Id: i, Ts: ts}
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %w", err)
	}
	bytes, err := protocol.Pack(body)
	if err != nil {
		return nil, fmt.Errorf("protocol.Pack failed: %w", err)
	}
	return bytes, nil
}

func readMsg(conn net.Conn) (public.Msg, error) {
	var msg public.Msg
	_, body, err := protocol.Read(conn)
	if err != nil {
		return msg, fmt.Errorf("protocol.Read failed: %w", err)
	}
	if err = json.Unmarshal(body, &msg); err != nil {
		return msg, fmt.Errorf("json.Unmarshal failed: %w", err)
	}
	return msg, nil
}

func parallelProcess[T any](s []T, goroutines int, do func(context.Context, int, int, []T) error) error {
	slen := len(s)
	goroutines = min(goroutines, slen)
	psize, remainder := max(1, slen/goroutines), slen%goroutines

	ctx, cancel := context.WithCancelCause(context.Background())
	var errOnce sync.Once
	var wg sync.WaitGroup
	var start, end int
	for range goroutines {
		start = end
		end = start + psize + remainder/max(remainder, 1)
		if remainder > 0 {
			remainder--
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			if err := do(ctx, start, end, s); err != nil {
				errOnce.Do(func() {
					cancel(err)
				})
				cancel(err)
			}
		}(start, min(end, slen))
	}

	wg.Wait()
	return context.Cause(ctx)
}

func avg(arr []int64) int64 {
	var sum int64
	for _, v := range arr {
		sum += v
	}
	return sum / int64(len(arr))
}
