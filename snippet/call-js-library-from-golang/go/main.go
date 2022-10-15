package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/dop251/goja"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body bytes.Buffer
		_, _ = io.Copy(&body, r.Body)

		sendMail(fillTemplate(body.String()))

		fmt.Fprintf(w, "OK")
	})

	log.Fatal(http.ListenAndServe(":8192", nil))
}

//go:embed goja/dist.js
var gojaJS string

const someServerValue = "some server value"

func fillTemplate(template string) string {
	vm := goja.New()

	_ = vm.Set("template", template)
	_ = vm.Set("serverValue", someServerValue)

	_, _ = vm.RunString(gojaJS)

	return vm.Get("result").String()
}

func sendMail(content string) {
	log.Printf("send mail: %s", content)
}
