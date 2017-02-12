package jsonapi

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Request struct {
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

	switch method {
	case "GET":
		reqObj.URL.RawQuery = string(reqJSON)
	case "POST":
		reqObj.Body = ioutil.NopCloser(bytes.NewReader(reqJSON))
	default:
		return nil, errors.New("JsonAPI unsupported request method")
	}

	return &Request{reqObj, reqJSON}, nil
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
	rspObj, err := client.Do(r.reqObj)
	if err != nil {
		return err
	}

	var data bytes.Buffer
	io.Copy(&data, rspObj.Body)
	rspObj.Body.Close()

	return json.Unmarshal(data.Bytes(), rsp)
}
