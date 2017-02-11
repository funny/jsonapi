package jsonapi

import (
	"crypto"
	"net/http"
	"testing"
)

func init() {
	api := New(crypto.SHA256, StdLogger)

	api.HandleFunc("/echo", func(ctx *Context) interface{} {
		var req map[string]int

		ctx.Request(&req)

		return map[string]int{
			"value_is": req["value"],
		}
	})

	api.HandleFunc("/verify", func(ctx *Context) interface{} {
		var req map[string]int

		ctx.Request(&req)

		ctx.Verify("123", 3)

		return map[string]int{
			"value_is": req["value"],
		}
	})

	go http.ListenAndServe(":8080", api)
}

func Test_Get(t *testing.T) {
	req, err := NewRequest("GET", "http://localhost:8080/echo", map[string]int{
		"value": 123,
	}, 0, "", 0)
	if err != nil {
		t.Fatal(err)
	}

	var rsp map[string]int
	err = Do(http.DefaultClient, req, &rsp)
	if err != nil {
		t.Fatal(err)
	}

	if rsp["value_is"] != 123 {
		t.Fatal(rsp)
	}
}

func Test_Post(t *testing.T) {
	req, err := NewRequest("POST", "http://localhost:8080/echo", map[string]int{
		"value": 123,
	}, 0, "", 0)
	if err != nil {
		t.Fatal(err)
	}

	var rsp map[string]int
	err = Do(http.DefaultClient, req, &rsp)
	if err != nil {
		t.Fatal(err)
	}

	if rsp["value_is"] != 123 {
		t.Fatal(rsp)
	}
}

func Test_VerifyGet(t *testing.T) {
	req, err := NewRequest("GET", "http://localhost:8080/verify", map[string]int{
		"value": 123,
	}, crypto.SHA256, "123", Now())
	if err != nil {
		t.Fatal(err)
	}

	var rsp map[string]int
	err = Do(http.DefaultClient, req, &rsp)
	if err != nil {
		t.Fatal(err)
	}

	if rsp["value_is"] != 123 {
		t.Fatal(rsp)
	}
}

func Test_VerifyPost(t *testing.T) {
	req, err := NewRequest("POST", "http://localhost:8080/verify", map[string]int{
		"value": 123,
	}, crypto.SHA256, "123", Now())
	if err != nil {
		t.Fatal(err)
	}

	var rsp map[string]int
	err = Do(http.DefaultClient, req, &rsp)
	if err != nil {
		t.Fatal(err)
	}

	if rsp["value_is"] != 123 {
		t.Fatal(rsp)
	}
}
