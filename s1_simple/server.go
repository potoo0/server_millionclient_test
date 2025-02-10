package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"server_millionclient/public"
	"server_millionclient/public/protocol"
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

	public.Logger.Info("listening", zap.String("addr", *addr))
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		public.Logger.Fatal("listen error", zap.Error(err))
	}

	go public.ServeMetrics()

	for {
		conn, err := ln.Accept()
		if err != nil {
			public.Logger.Error("tcp accept failed", zap.Error(err))

			if errors.Is(err.(*net.OpError).Err, net.ErrClosed) {
				return
			}
			continue
		}

		go func() {
			public.ConnectionCount.Inc()
			for {
				if err := handleConn(conn); err != nil {
					if !errors.Is(err, io.EOF) {
						public.Logger.Info("handle conn failed", zap.Error(err))
					}
					public.ConnectionCount.Dec()
					_ = conn.Close()
					break
				}
			}
		}()
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
