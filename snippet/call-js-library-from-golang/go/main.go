package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body bytes.Buffer
		_, _ = io.Copy(&body, r.Body)

		sendMail(body.String())

		fmt.Fprintf(w, "OK")
	})

	log.Fatal(http.ListenAndServe(":8192", nil))
}

func sendMail(content string) {
	log.Printf("send mail: %s", content)
}
