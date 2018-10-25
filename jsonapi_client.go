package jsonapi

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	r.reqObj.Header.Set("content-type", "application/json")

	switch r.method {
	case "GET":
		r.reqObj.URL.RawQuery = url.QueryEscape(string(r.reqJSON))
	case "POST":
		r.reqObj.ContentLength = int64(len(r.reqJSON))
		r.reqObj.Body = ioutil.NopCloser(bytes.NewReader(r.reqJSON))
	default:
		return errors.New("JsonAPI unsupported request method")
	}

	rspObj, err := client.Do(r.reqObj)
	if err != nil {
		return err
	}

	if rspObj.StatusCode == http.StatusInternalServerError {
		rsp = new(JsonAPIError)
	}

	err = json.NewDecoder(rspObj.Body).Decode(rsp)
	rspObj.Body.Close()

	if err != nil {
		return err
	}

	switch rspObj.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusInternalServerError:
		if e, ok := rsp.(*JsonAPIError); ok {
			return fmt.Errorf("internal server error : %s", e.Err)
		}
		return errors.New("unknow error")
	default:
		return fmt.Errorf("unknow error code %d", rspObj.StatusCode)
	}
}
