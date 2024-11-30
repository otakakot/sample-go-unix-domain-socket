package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	listener, err := net.Listen("unix", "socket")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			slog.InfoContext(context.Background(), "listener close error: "+err.Error())
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	go func() {
		for {
			slog.Info("start accept")

			conn, err := listener.Accept()
			if err != nil && !errors.Is(err, net.ErrClosed) {
				panic(err)
			}

			go func() {
				if conn == nil {
					return
				}

				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					panic(err)
				}

				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					panic(err)
				}

				slog.InfoContext(req.Context(), "request: "+string(dump))

				res := http.Response{
					StatusCode: http.StatusOK,
					ProtoMajor: 1,
					ProtoMinor: 0,
					Body:       io.NopCloser(strings.NewReader("Hello, World from unix domain socket!")),
				}

				res.Write(conn)

				if err := conn.Close(); err != nil {
					slog.ErrorContext(context.Background(), "conn close error: "+err.Error())
				}
			}()
		}
	}()

	<-ctx.Done()

	os.Remove("socket")

	slog.Info("shutdown")
}
