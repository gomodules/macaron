// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Package macaron is a high productive and modular design web framework in Go.
package macaron

import (
	"io"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/Unknwon/macaron/inject"
)

func Version() string {
	return "0.1.5.0830"
}

// Handler can be any callable function.
// Macaron attempts to inject services into the handler's argument list,
// and panics if an argument could not be fullfilled via dependency injection.
type Handler interface{}

// validateHandler makes sure a handler is a callable function,
// and panics if it is not.
func validateHandler(handler Handler) {
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		panic("mocaron handler must be a callable function")
	}
}

// Macaron represents the top level web application.
// inject.Injector methods can be invoked to map services on a global level.
type Macaron struct {
	inject.Injector
	handlers []Handler
	action   Handler
	*Router
	logger *log.Logger
}

// NewWithLogger creates a bare bones Macaron instance.
// Use this method if you want to have full control over the middleware that is used.
// You can specify logger output writer with this function.
func NewWithLogger(out io.Writer) *Macaron {
	m := &Macaron{
		Injector: inject.New(),
		action:   func() {},
		Router:   NewRouter(),
		logger:   log.New(out, "[Macaron] ", 0),
	}
	m.Router.m = m
	m.Map(m.logger)
	m.Map(defaultReturnHandler())
	m.notFound = func(resp http.ResponseWriter, req *http.Request) {
		c := m.createContext(resp, req)
		c.handlers = append(c.handlers, http.NotFound)
		c.run()
	}
	return m
}

// New creates a bare bones Macaron instance.
// Use this method if you want to have full control over the middleware that is used.
func New() *Macaron {
	return NewWithLogger(os.Stdout)
}

// Classic creates a classic Macaron with some basic default middleware:
// mocaron.Logger, mocaron.Recovery and mocaron.Static.
func Classic() *Macaron {
	m := New()
	m.Use(Logger())
	m.Use(Recovery())
	m.Use(Static("public"))
	return m
}

// Handlers sets the entire middleware stack with the given Handlers.
// This will clear any current middleware handlers,
// and panics if any of the handlers is not a callable function
func (m *Macaron) Handlers(handlers ...Handler) {
	m.handlers = make([]Handler, 0)
	for _, handler := range handlers {
		m.Use(handler)
	}
}

// Action sets the handler that will be called after all the middleware has been invoked.
// This is set to macaron.Router in a macaron.Classic().
func (m *Macaron) Action(handler Handler) {
	validateHandler(handler)
	m.action = handler
}

// Use adds a middleware Handler to the stack,
// and panics if the handler is not a callable func.
// Middleware Handlers are invoked in the order that they are added.
func (m *Macaron) Use(handler Handler) {
	validateHandler(handler)
	m.handlers = append(m.handlers, handler)
}

func (m *Macaron) createContext(rw http.ResponseWriter, req *http.Request) *Context {
	c := &Context{
		Injector: inject.New(),
		handlers: m.handlers,
		action:   m.action,
		index:    0,
		Router:   m.Router,
		Req:      req,
		Resp:     NewResponseWriter(rw),
		Data:     make(map[string]interface{}),
	}
	c.SetParent(m)
	c.Map(c)
	c.MapTo(c.Resp, (*http.ResponseWriter)(nil))
	c.Map(req)
	return c
}

// ServeHTTP is the HTTP Entry point for a Macaron instance.
// Useful if you want to control your own HTTP server.
// Be aware that none of middleware will run without registering any router.
func (m *Macaron) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	m.Router.ServeHTTP(resp, req)
}

// getDefaultListenAddr returns default server listen address of Macaron.
func getDefaultListenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	host := os.Getenv("HOST")
	if len(host) == 0 {
		host = "0.0.0.0"
	}
	return host + ":" + port
}

// RunOnAddr runs server in given address and port.
func (m *Macaron) RunOnAddr(addr string) {
	logger := m.Injector.GetVal(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
	logger.Printf("listening on %s (%s)\n", addr, Env)
	logger.Fatalln(http.ListenAndServe(addr, m))
}

// Run the http server. Listening on os.GetEnv("PORT") or 4000 by default.
func (m *Macaron) Run() {
	m.RunOnAddr(getDefaultListenAddr())
}

// \_   _____/ _______  __
//  |    __)_ /    \  \/ /
//  |        \   |  \   /
// /_______  /___|  /\_/
//         \/     \/

const (
	DEV  string = "development"
	PROD string = "production"
	TEST string = "test"
)

// Env is the environment that Macaron is executing in.
// The MACARON_ENV is read on initialization to set this variable.
var Env = DEV
var Root string

func setENV(e string) {
	if len(e) > 0 {
		Env = e
	}
}

func init() {
	setENV(os.Getenv("MACARON_ENV"))
	var err error
	Root, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}
