package jsonapi

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var StdLogger = NewDefaultLogger(log.New(os.Stderr, "", log.LstdFlags))

func Now() int {
	return int(time.Now().UTC().Unix())
}

type Logger interface {
	Debug(req *http.Request)
	Panic(req *http.Request, err interface{})
	Fatal(req *http.Request, msg string, err error)
}

var _ Logger = (*DefaultLogger)(nil)

type DefaultLogger struct {
	logger  *log.Logger
	IsDebug bool
}

func NewDefaultLogger(logger *log.Logger) *DefaultLogger {
	return &DefaultLogger{logger: logger, IsDebug: false}
}

func (l *DefaultLogger) Debug(r *http.Request) {
	if l.IsDebug {
		l.logger.Printf("[DEBUG] %s %s", r.Method, r.RequestURI)
	}
}

func (l *DefaultLogger) Panic(r *http.Request, err interface{}) {
	l.logger.Printf("[PANIC] %s %s JsonAPI recover a panic: %v", r.Method, r.RequestURI, err)
}

func (l *DefaultLogger) Fatal(r *http.Request, msg string, err error) {
	if err != nil {
		l.logger.Printf("[FATAL] %s %s %s: %s", r.Method, r.RequestURI, msg, err)
	} else {
		l.logger.Printf("[FATAL] %s %s: %s", r.Method, r.RequestURI, msg)
	}
}

type Handler interface {
	ServeJSON(ctx *Context) interface{}
}

var _ Handler = (HandlerFunc)(nil)

type HandlerFunc func(ctx *Context) interface{}

func (f HandlerFunc) ServeJSON(ctx *Context) interface{} {
	ctx.logger.Debug(ctx.request)
	return f(ctx)
}

var _ http.Handler = (*API)(nil)

type API struct {
	mux    *http.ServeMux
	hash   crypto.Hash
	logger Logger
}

var gAPI *API

func New(hash crypto.Hash, logger Logger) *API {
	gAPI = &API{
		hash:   hash,
		logger: logger,
		mux:    http.NewServeMux(),
	}
	return gAPI
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mux.ServeHTTP(w, r)
}

func (api *API) HandleFunc(path string, handler func(ctx *Context) interface{}) {
	api.Handle(path, HandlerFunc(handler))
}

func (api *API) Handle(path string, handler Handler) {
	api.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil && err != FatalError {
				api.logger.Panic(r, err)
				http.Error(w, `{"error":"unknow"}`, 500)
			}
		}()

		ctx := Context{hash: api.hash, logger: api.logger, response: w, request: r}

		rsp := handler.ServeJSON(&ctx)

		data, err := json.Marshal(rsp)
		if err != nil {
			ctx.Fatal("JSON marshal failed", err)
		}
		w.Write(data)
	})
}

var FatalError = errors.New("JsonAPI Fatal")

type Context struct {
	msg      []byte
	hash     crypto.Hash
	logger   Logger
	request  *http.Request
	response http.ResponseWriter
}

func (ctx *Context) Fatal(msg string, err ...error) {
	if len(err) != 0 {
		ctx.logger.Fatal(ctx.request, msg, err[0])
	} else {
		ctx.logger.Fatal(ctx.request, msg, nil)
	}
	http.Error(ctx.response, `{"error":"`+msg+`"}`, 500)
	panic(FatalError)
}

func (ctx *Context) Request(req interface{}) {
	switch ctx.request.Method {
	case "GET":
		query, err := url.QueryUnescape(ctx.request.URL.RawQuery)
		if err != nil {
			ctx.Fatal("Invalid query string", err)
		}
		ctx.msg = []byte(query)
	case "POST":
		var buf bytes.Buffer
		_, err := io.Copy(&buf, ctx.request.Body)
		if err != nil {
			ctx.Fatal("Bad request", err)
		}
		ctx.request.Body.Close()
		ctx.msg = buf.Bytes()
	}

	if len(ctx.msg) == 0 {
		return
	}

	err := json.Unmarshal(ctx.msg, req)
	if err != nil {
		ctx.Fatal(fmt.Sprintf("JSON unmarshal failed, msg: %s", string(ctx.msg)), err)
	}
}

func (ctx *Context) HttpRequest() *http.Request {
	return ctx.request
}

func (ctx *Context) HttpResponseWriter() http.ResponseWriter {
	return ctx.response
}

func (ctx *Context) Verify(key string, timeout int) {
	timeStr := ctx.request.Header.Get("t")

	if timeout > 0 {
		timeVal, err := strconv.Atoi(timeStr)
		if err != nil {
			ctx.Fatal("HTTP header 't' not a number", err)
		}
		if timeVal+timeout < Now() {
			ctx.Fatal("Request expired")
		}
	}

	if key != "" {
		if ctx.msg == nil {
			ctx.Fatal("ctx.msg == nil")
		}
		sigHead := ctx.request.Header.Get("s")
		sigData, err := base64.StdEncoding.DecodeString(sigHead)
		if err != nil {
			ctx.Fatal("HTTP header 's' not base64 string", err)
		}
		sigGood := signature(
			ctx.hash,
			[]byte(key),
			[]byte(timeStr),
			[]byte(ctx.request.URL.Path),
			ctx.msg,
		)
		if !bytes.Equal([]byte(sigData), sigGood) {
			ctx.Fatal("Signature validate failed")
		}
	}
}

func signature(hash crypto.Hash, key, time, path, msg []byte) []byte {
	h := hash.New()
	h.Write(key)
	h.Write(time)
	h.Write(path)
	h.Write(msg)
	return h.Sum(nil)
}
