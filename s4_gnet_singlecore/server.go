package main

import (
	"encoding/json"
	"flag"
	"server_millionclient/public"
	"server_millionclient/public/protocol"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
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

	go public.ServeMetrics()

	public.Logger.Info("listening", zap.String("addr", *addr))
	p := goroutine.Default()
	defer p.Release()
	eventHandler := &server{pool: p}
	addrFull := *addr
	if addrFull[0] == ':' {
		addrFull = "0.0.0.0" + *addr
	}
	if err := gnet.Run(eventHandler, "tcp://"+addrFull, gnet.WithMulticore(false), gnet.WithLogger(public.Logger.Sugar())); err != nil {
		public.Logger.Fatal("gnet.Run failed", zap.Error(err))
	}
}

func (s *server) OnOpen(_ gnet.Conn) (out []byte, action gnet.Action) {
	public.ConnectionCount.Inc()
	return
}

func (s *server) OnTraffic(conn gnet.Conn) (action gnet.Action) {
	defer public.RequestCount.Inc()
	_, body, err := protocol.Read(conn)
	if err != nil {
		public.Logger.Info("read failed", zap.Error(err))
		return gnet.Close
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
		public.Logger.Info("pack failed", zap.Error(err))
		return gnet.Close
	} else {
		if err := conn.AsyncWrite(bytes, func(c gnet.Conn, err error) error {
			if err != nil {
				public.Logger.Info("write failed", zap.Error(err))
			}
			return err
		}); err != nil {
			public.Logger.Info("async write failed", zap.Error(err))
			return gnet.Close
		}
	}
	return
}

func (s *server) OnClose(_ gnet.Conn, _ error) (action gnet.Action) {
	public.ConnectionCount.Dec()
	return
}

type server struct {
	gnet.BuiltinEventEngine

	pool *goroutine.Pool
	eng  gnet.Engine
}

var _ gnet.EventHandler = (*server)(nil)
