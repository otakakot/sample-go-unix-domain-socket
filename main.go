package main

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	os.Remove("socket")

	port := cmp.Or(os.Getenv("PORT"), "8080")

	hdl := http.NewServeMux()

	hdl.HandleFunc("/", Hanele)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           hdl,
		ReadHeaderTimeout: 30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	go func() {
		slog.Info("start server listen")

		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	listener, err := net.Listen("unix", "socket")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			slog.InfoContext(context.Background(), "listener close error: "+err.Error())
		}
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}

			go func() {
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
					StatusCode: 200,
					ProtoMajor: 1,
					ProtoMinor: 0,
					Body:       io.NopCloser(strings.NewReader("Hello, World!")),
				}

				res.Write(conn)

				if err := conn.Close(); err != nil {
					slog.InfoContext(context.Background(), "conn close error: "+err.Error())
				}
			}()
		}
	}()

	<-ctx.Done()

	slog.Info("start server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := srv.Shutdown(ctx); !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(context.Background(), "server shutdown error: "+err.Error())
	}

	slog.Info("done server shutdown")

}

func Hanele(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial("unix", "socket")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	req.Write(conn)

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	slog.InfoContext(r.Context(), "response: "+string(dump))

	defer func() {
		if err := conn.Close(); err != nil {
			slog.InfoContext(r.Context(), "conn close error: "+err.Error())
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := w.Write(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}
