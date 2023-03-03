package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var requestTimeout = 400 * time.Millisecond

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	fileTxt, err := os.Create("cotacao.txt")
	if err != nil {
		panic(err)
	}

	responseJson, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	rate := "Dólar: {" + string(responseJson) + "}"

	io.Copy(fileTxt, strings.NewReader(rate))
}
