package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
)

type LoggingRoundTripper struct {
	Transport http.RoundTripper
}

func (l *LoggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	log.Printf("Sending request to %s\n", r.Host)
	resp, err := l.Transport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	log.Printf("Got response from %s, status=%d\n", r.Host, resp.StatusCode)
	return resp, nil
}

func TestRoundTripper(t *testing.T) {
	client := &http.Client{
		Transport: &LoggingRoundTripper{
			Transport: http.DefaultTransport,
		},
	}

	resp, _ := client.Get("http://localhost:9000/ping")
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	fmt.Println(string(data))
}
