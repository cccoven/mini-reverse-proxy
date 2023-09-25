package main

import (
	"fmt"
	"log"
	"net/http"
)

type Handler struct {
	Transport http.RoundTripper
}

func (h *Handler) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println("Receive a request from ", r.RemoteAddr)
	resp, err := h.Transport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	
}

func main() {
	handler := &Handler{
		Transport: http.DefaultTransport,
	}

	log.Println(http.ListenAndServe(":8080", handler))
}
