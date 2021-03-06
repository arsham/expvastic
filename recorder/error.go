// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package recorder

import (
	"fmt"
	"time"
)

// Recorder related errors.
// ErrEmptyEndpoint is returned when the given endpoint is empty.
// ErrPingNotCalled is returned if the caller calls the record without pinging.
var (
	ErrEmptyName      = fmt.Errorf("name cannot be empty")
	ErrEmptyEndpoint  = fmt.Errorf("endpoint cannot be empty")
	ErrEmptyIndexName = fmt.Errorf("index_name cannot be empty")
	ErrPingNotCalled  = fmt.Errorf("the caller forgot to ask me pinging")
	ErrNillLogger     = fmt.Errorf("nil logger")
)

// InvalidEndpointError is returned when the endpoint is not a valid URL.
type InvalidEndpointError string

func (e InvalidEndpointError) Error() string {
	return fmt.Sprintf("invalid endpoint: %s", string(e))
}

// ParseTimeOutError is returned when the timeout cannot be parsed.
type ParseTimeOutError struct {
	Timeout string
	Err     error
}

func (e ParseTimeOutError) Error() string {
	return fmt.Sprintf("parse timeout (%s): %s", e.Timeout, e.Err)
}

// EndpointNotAvailableError is returned when the endpoint is not available.
type EndpointNotAvailableError struct {
	Endpoint string
	Err      error
}

func (e EndpointNotAvailableError) Error() string {
	return fmt.Sprintf("endpoint (%s) not available: %s", e.Endpoint, e.Err)
}

// InvalidIndexNameError is returned when the index name is invalid.
type InvalidIndexNameError string

func (e InvalidIndexNameError) Error() string {
	return fmt.Sprintf("Index name (%s) is not valid", string(e))
}

// LowTimeout is returned when the interval is zero.
type LowTimeout time.Duration

func (e LowTimeout) Error() string {
	return fmt.Sprintf("timeout should be more than 1 second: %d", e)
}
