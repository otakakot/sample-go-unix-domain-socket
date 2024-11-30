package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
)

func main() {
	conn, err := net.Dial("unix", "socket")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"",
		nil,
	)
	if err != nil {
		panic(err)
	}

	req.Write(conn)

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		panic(err)
	}

	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		panic(err)
	}

	slog.Info("request: " + string(dump))

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	slog.Info("response: " + string(body))
}
