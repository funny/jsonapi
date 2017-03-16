package jsonapi

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type Request struct {
	method  string
	reqObj  *http.Request
	reqJSON []byte
}

func Get(urlStr string, req interface{}) (*Request, error) {
	return newRequest("GET", urlStr, req)
}

func Post(urlStr string, req interface{}) (*Request, error) {
	return newRequest("POST", urlStr, req)
}

func newRequest(method, urlStr string, req interface{}) (*Request, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	reqObj, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	return &Request{method, reqObj, reqJSON}, nil
}

func (r *Request) Signature(hash crypto.Hash, key string, time int) {
	timeStr := strconv.Itoa(time)
	sigData := signature(
		hash,
		[]byte(key),
		[]byte(timeStr),
		[]byte(r.reqObj.URL.Path),
		r.reqJSON,
	)
	sigHead := base64.StdEncoding.EncodeToString(sigData)
	r.reqObj.Header.Set("t", timeStr)
	r.reqObj.Header.Set("s", sigHead)
}

func (r *Request) Do(client *http.Client, rsp interface{}) error {
	switch r.method {
	case "GET":
		r.reqObj.URL.RawQuery = url.QueryEscape(string(r.reqJSON))
	case "POST":
		r.reqObj.Body = ioutil.NopCloser(bytes.NewReader(r.reqJSON))
	default:
		return errors.New("JsonAPI unsupported request method")
	}

	rspObj, err := client.Do(r.reqObj)
	if err != nil {
		return err
	}

	err = json.NewDecoder(rspObj.Body).Decode(rsp)
	rspObj.Body.Close()
	return err
}
