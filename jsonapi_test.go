package jsonapi

import (
	"bytes"
	"crypto"
	"io"
	"net/http"
	"testing"
)

func Test_Echo(t *testing.T) {
	api := New(crypto.SHA256, StdLogger)

	api.HandleFunc("/api", func(ctx *Context) interface{} {
		var req map[string]interface{}

		ctx.Request(&req)

		return map[string]interface{}{
			"value_is": req["value"],
		}
	})

	go http.ListenAndServe(":8080", api)

	rsp, err := http.DefaultClient.Get(`http://localhost:8080/api?{"value":123}`)
	if err != nil {
		t.Fatal(err)
	}

	var data bytes.Buffer
	io.Copy(&data, rsp.Body)
	rsp.Body.Close()

	if data.String() != `{"value_is":123}` {
		t.Fatal(data.String())
	}
}
