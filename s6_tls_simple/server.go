package main

import (
	"crypto/tls"
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
	cer, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		public.Logger.Fatal("load certificate file error", zap.Error(err))
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen("tcp", *addr, config)
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

// go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host 1m-server
var (
	certBytes = []byte(`-----BEGIN CERTIFICATE-----
MIIDBTCCAe2gAwIBAgIQQB8UAlRvHS6L3dGaeEKagzANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMB4XDTI1MDIxMjE0MTQxN1oXDTI2MDIxMjE0MTQx
N1owEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAKrAH2tUoKCJuLt6PT+83NS0xa/6xCAqCeSMtUfRYCh5RrE+Ee9sWbNp
bis9QUsXTzAvXCk0X1ajsO2PKU4MFqVLN1Ks9A3yQFFPPXlNzqgFlURI7NzSR/ad
OSGH6Txic+ehjccM9A+LMlI2KdU9+PNAPKEoEH/i3n+HlaO6sOl4kMS3nB2nlrSk
dtPZab8UfI6YnSzOqL+d9/fOUMVdbmvKk8bGXRi7X1B3jUc5ehT9O9uBipn3MLUr
EN3fBCJu+JmH2cCn1Skjw1L9uQqpoUslyIyKcYG64hxPNYb6Q2ZQi41hzlCC88d1
TcaEBS+K84uzRSDAbQSkhTGFF5Ahd4cCAwEAAaNXMFUwDgYDVR0PAQH/BAQDAgWg
MBMGA1UdJQQMMAoGCCsGAQUFBwMBMAwGA1UdEwEB/wQCMAAwIAYDVR0RBBkwF4IJ
MW0tc2VydmVyhwR/AAABhwQKWQMKMA0GCSqGSIb3DQEBCwUAA4IBAQAehnUHAZ8/
5pGMK6Afo8E4HmBIxUv8dJnji/B3d2JXYqcGSnTmOpA48Bk+zFDe4lYNk4vxb9Sr
DbPm7hzsYodIfRUlV7wtGCvKuo6uyHJmI2t0xZcd3Gw9Du2CvwP2zRfFDdQ9EYZO
KsTw6dz/OEwEj+aCxhpbks3IjnQc4bmu1a43pOCg3UIZdhqdDhS0xP+bdLvhOzFc
6AniPj5kTo3QTQDn2cDKF1xXiTEA8rWCZPEeL75bD79FH1en3giZm+GedgRAqPKN
IXhuCjACrVUDHkt8kOVXp6aP6z0A6+9XAEvmiJZxG7rLLwy5XNmBFnJuC5MzUqo4
HNV5j5L+pyx6
-----END CERTIFICATE-----`)
	keyBytes = []byte(`-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCqwB9rVKCgibi7
ej0/vNzUtMWv+sQgKgnkjLVH0WAoeUaxPhHvbFmzaW4rPUFLF08wL1wpNF9Wo7Dt
jylODBalSzdSrPQN8kBRTz15Tc6oBZVESOzc0kf2nTkhh+k8YnPnoY3HDPQPizJS
NinVPfjzQDyhKBB/4t5/h5WjurDpeJDEt5wdp5a0pHbT2Wm/FHyOmJ0szqi/nff3
zlDFXW5rypPGxl0Yu19Qd41HOXoU/TvbgYqZ9zC1KxDd3wQibviZh9nAp9UpI8NS
/bkKqaFLJciMinGBuuIcTzWG+kNmUIuNYc5QgvPHdU3GhAUvivOLs0UgwG0EpIUx
hReQIXeHAgMBAAECggEAKZ5mihy4gijPdDLZVv3LvbTKMpim0Ugt3R1G2lh4XaUh
y/XbHUaFnqtmBPgLQChQTuhcSFbRniaL63tnj/R2WJe6xlYNrpCLiMYNr9F9O4sQ
1PIJedFvZPbxg/DCsss0gRLpocjQfDFrdIprK+TNF01i+czwKJu9q8v6d0v77wvj
8wlqf2R+O/pJQzQKY56n0M+R/vOJLI2Ms7ADwHVB5Gly6Wg7qW8u9imTt0C5/fN1
PvRnDExKTB3UpC60gPApIGxmSWM0v7pVNn9g8ENBuP0/z8co/ZE/PTmllpR8z8to
bSw4pLAtJI/k3gV1EJZQIk2bMle1tdMPXaPSwBzX4QKBgQDd8XbAwgqrDAGN5DRc
88nRPRWKSVdU9AL7KXAdvXlJ4GkVvOB2ZYv+0Y4PqnJ2ezWb3fZJ3JL+U+KVC+py
k4mhrdoS8+Y8jNq9XgoX/i8B3P965VkkdjKVrliWkw0tZ9gsLjhkWqyc8dTaE0UD
Fb3i7H3pC9kpNBJ+PJJT44623wKBgQDE86sNidknAvucNiukqY5fTKZh20Su2pgd
Yh3ZBV/8hEacid0aDX6KC7VP1CgECd/u7ZZwY9k1nFpjfg2vuZQeVt5bLB6LhK11
nO1bHemF8eZxV7hKwfgVUw62BtXsCIfyQdLgUGc/11991HqbkMGcW+9uGIK/q+P3
HKRLamycWQKBgQDUlTUO2o2HWl+eviedpPD5Fs4r/6XDvFmiowU9pz+mkGl3JcvF
++wE7klpHfS3IbquigMeqkStkEGmS5yLlF+u2ivYHX+5HZ1i5tE6PABgg4K9/zHM
J9652h4GU+G6TQ4U+0yOav+M8GHVY8Gle8y+r5DGiM+/lJ3mBjSOX5dR9wKBgFY8
u3E6IrNKQxGrRoDbHVPtJA1FDVXisShshdU43UacRK7WTtHRhs67QbCqnLrn9/2O
Wojrr3gh9hIKZ8PB5nFCaCpTryw39BvDksqK1m2n9dc7KZ7SP+ZWb+KUK6cmNSCG
YeeGTS9PBqj6GJV1VNE6ECSM5vM2OKNDD01WVChBAoGACJDJAyQU33YBaS4DS9Bq
AZAAbK2re9zptMdJQzw0mNddQqZEjOziHZLUKBxkdb146gnkf592zlRV+YrOnrl7
TLv/gciglk/ZjUN2PZNpeqvtx+6q6h72vu54vim2GmPL26C6Z9rRU9LX8vTTdUBK
MzvBCiBbSGIVXqtRAVLgBgM=
-----END PRIVATE KEY-----`)
)
