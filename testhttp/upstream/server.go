package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Receive a request from ", r.RemoteAddr)
		w.Write([]byte("pong\n"))
	})

	log.Println(http.ListenAndServe(":9000", nil))
}
