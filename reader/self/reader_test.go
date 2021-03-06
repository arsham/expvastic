// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package self_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arsham/expipe/reader"
	"github.com/arsham/expipe/reader/self"
	rt "github.com/arsham/expipe/reader/testing"
)

func getTestServer() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{}"))
		}),
	)
}

type Construct struct {
	*rt.BaseConstruct
	testServer *httptest.Server
}

func (c *Construct) TestServer() *httptest.Server {
	c.testServer = getTestServer()
	return c.testServer
}

func (c *Construct) Object() (reader.DataReader, error) {
	s, err := self.New(c.Setters()...)
	if err == nil {
		s.SetTestMode()
	}
	return s, err
}

func TestSelfReader(t *testing.T) {
	rt.TestSuites(t, func() (rt.Constructor, func()) {
		c := &Construct{
			testServer:    getTestServer(),
			BaseConstruct: rt.NewBaseConstruct(),
		}
		return c, func() { c.testServer.Close() }
	})
}

func TestWithTempServer(t *testing.T) {
	t.Parallel()
	var r reader.Constructor
	r = new(rt.Reader)
	err := self.WithTempServer()(r)
	if err == nil {
		t.Error("WithTempServer(): err = (nil); want (error)")
	}
	r = new(self.Reader)
	err = self.WithTempServer()(r)
	if err != nil {
		t.Errorf("WithTempServer(): err = (%#v); want (nil)", err)
	}
	s := r.(*self.Reader)
	reader.WithTimeout(time.Second)(s)
	err = s.Ping()
	if err != nil {
		t.Errorf("Ping(): err = (%#v); want (nil)", err)
	}
}
