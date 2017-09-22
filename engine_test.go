// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package expipe_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arsham/expipe"
	"github.com/arsham/expipe/internal"
	"github.com/arsham/expipe/internal/token"
	"github.com/arsham/expipe/reader"
	reader_testing "github.com/arsham/expipe/reader/testing"
	"github.com/arsham/expipe/recorder"
	recorder_testing "github.com/arsham/expipe/recorder/testing"

	"github.com/pkg/errors"
)

// TODO: test engine closes readers when recorder goes out of scope

var (
	log        internal.FieldLogger
	testServer *httptest.Server
)

func TestMain(m *testing.M) {
	log = internal.DiscardLogger()
	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	exitCode := m.Run()
	testServer.Close()
	os.Exit(exitCode)
}

func TestNewWithReadRecorder(t *testing.T) {
	ctx := context.Background()

	rec, err := recorder_testing.New(ctx, log, "a", testServer.URL, "aa", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	red, err := reader_testing.New(log, testServer.URL, "a", "dd", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	red2, err := reader_testing.New(log, testServer.URL, "a", "dd", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red.Name(): red, red2.Name(): red2})
	if err != nil {
		t.Errorf("want (nil), got (%v)", err)
	}
	if e == nil {
		t.Error("want Engine, got nil")
	}
}

func TestEngineSendJob(t *testing.T) {
	var recorderID token.ID
	ctx, cancel := context.WithCancel(context.Background())

	red, err := reader_testing.New(log, testServer.URL, "reader_example", "example_type", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	red.ReadFunc = func(job *token.Context) (*reader.Result, error) {
		recorderID = token.NewUID()
		resp := &reader.Result{
			ID:       recorderID,
			Content:  []byte(`{"devil":666}`),
			TypeName: red.TypeName(),
			Mapper:   red.Mapper(),
		}
		return resp, nil
	}

	rec, err := recorder_testing.New(ctx, log, "recorder_example", testServer.URL, "intexName", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	rec.RecordFunc = func(ctx context.Context, job *recorder.Job) error {
		if job.ID != recorderID {
			t.Errorf("want (%d), got (%s)", recorderID, job.ID)
		}
		return nil
	}

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red.Name(): red})
	if err != nil {
		t.Errorf("want (nil), got (%v)", err)
	}
	done := make(chan struct{})
	go func() {
		e.Start()
		done <- struct{}{}
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("expected the engine to quit gracefully")
	}
}

func TestEngineMultiReader(t *testing.T) {
	count := 10
	ctx, cancel := context.WithCancel(context.Background())
	IDs := make([]string, count)
	idChan := make(chan token.ID)
	for i := 0; i < count; i++ {
		id := token.NewUID()
		IDs[i] = id.String()
		go func(id token.ID) {
			idChan <- id
		}(id)
	}

	rec, err := recorder_testing.New(ctx, log, "recorder_example", testServer.URL, "intexName", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	rec.RecordFunc = func(ctx context.Context, job *recorder.Job) error {
		if !internal.StringInSlice(job.ID.String(), IDs) {
			t.Errorf("want once of (%s), got (%s)", strings.Join(IDs, ","), job.ID)
		}
		return nil
	}

	reds := make(map[string]reader.DataReader, count)
	for i := 0; i < count; i++ {

		name := fmt.Sprintf("reader_example_%d", i)
		red, err := reader_testing.New(log, testServer.URL, name, "example_type", time.Hour, time.Hour, 5)
		if err != nil {
			t.Fatal(err)
		}
		red.ReadFunc = func(job *token.Context) (*reader.Result, error) {
			resp := &reader.Result{
				ID:       <-idChan,
				Content:  []byte(`{"devil":666}`),
				TypeName: red.TypeName(),
				Mapper:   red.Mapper(),
			}
			return resp, nil
		}
		reds[red.Name()] = red
	}

	e, err := expipe.New(ctx, log, rec, reds)
	if err != nil {
		t.Errorf("want (nil), got (%v)", err)
	}
	done := make(chan struct{})
	go func() {
		e.Start()
		done <- struct{}{}
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("expected the engine to quit gracefully")
	}
}

func TestEngineNewWithConfig(t *testing.T) {
	ctx := context.Background()

	red, err := reader_testing.NewConfig("", "reader_example", log, "nowhere", "/still/nowhere", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := recorder_testing.NewConfig("recorder_example", log, "nowhere", time.Hour, 5, "index")
	if err != nil {
		t.Fatal(err)
	}

	e, err := expipe.WithConfig(ctx, log, rec, red)
	if errors.Cause(err) != reader.ErrEmptyName {
		t.Errorf("want ErrEmptyReaderName, got (%v)", err)
	}
	if e != nil {
		t.Errorf("want nil, got (%v)", e)
	}

	// triggering recorder errors
	rec, _ = recorder_testing.NewConfig("recorder_example", log, "nowhere", time.Hour, 5, "index")
	red, _ = reader_testing.NewConfig("same_name_is_illegal", "reader_example", log, testServer.URL, "/still/nowhere", time.Hour, time.Hour, 5)

	e, err = expipe.WithConfig(ctx, log, rec, red)
	if e != nil {
		t.Errorf("want nil, got (%v)", e)
	}
	if _, ok := errors.Cause(err).(recorder.ErrInvalidEndpoint); !ok {
		t.Errorf("want ErrInvalidEndpoint, got (%v)", err)
	}

	red, _ = reader_testing.NewConfig("same_name_is_illegal", "reader_example", log, testServer.URL, "/still/nowhere", time.Hour, time.Hour, 5)
	red2, _ := reader_testing.NewConfig("same_name_is_illegal", "reader_example", log, testServer.URL, "/still/nowhere", time.Hour, time.Hour, 5)
	rec, _ = recorder_testing.NewConfig("recorder_example", log, testServer.URL, time.Hour, 5, "index")
	e, err = expipe.WithConfig(ctx, log, rec, red, red2)
	if err != nil {
		t.Errorf("want nil, got (%v)", err)
	}
	if e == nil {
		t.Errorf("want Engine, got (%v)", e)
	}
}

func TestEngineErrorsIfReaderNotPinged(t *testing.T) {
	ctx := context.Background()
	redServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	recServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer recServer.Close()
	redServer.Close() // making sure no one else is got this random port at this time

	rec, err := recorder_testing.New(ctx, log, "a", recServer.URL, "aa", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	red, err := reader_testing.New(log, redServer.URL, "a", "dd", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red.Name(): red})
	if err == nil {
		t.Error("want ErrPing, got nil")
	}

	if _, ok := errors.Cause(err).(expipe.ErrPing); !ok {
		t.Errorf("want ErrPing, got (%v)", err)
	}
	if e != nil {
		t.Errorf("want (nil), got (%v)", e)
	}
}

func TestEngineErrorsIfRecorderNotPinged(t *testing.T) {
	ctx := context.Background()
	redServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	recServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	recServer.Close() // making sure no one else is got this random port at this time
	defer redServer.Close()

	rec, err := recorder_testing.New(ctx, log, "a", recServer.URL, "aa", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	red, err := reader_testing.New(log, redServer.URL, "a", "dd", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red.Name(): red})
	if err == nil {
		t.Error("want ErrPing, got nil")
	}

	if _, ok := errors.Cause(err).(expipe.ErrPing); !ok {
		t.Errorf("want ErrPing, got (%v)", err)
	}
	if e != nil {
		t.Errorf("want (nil), got (%v)", e)
	}
}

func TestEngineOnlyErrorsIfAllReadersNotPinged(t *testing.T) {
	ctx := context.Background()
	deadServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	liveServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer liveServer.Close()
	deadServer.Close() // making sure no one else is got this random port at this time

	rec, err := recorder_testing.New(ctx, log, "a", liveServer.URL, "aa", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	red1, err := reader_testing.New(log, liveServer.URL, "a", "dd", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	red2, err := reader_testing.New(log, deadServer.URL, "b", "ddb", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red1.Name(): red1, red2.Name(): red2})
	if err != nil {
		t.Errorf("want nil, got (%v)", err)
	}

	if e == nil {
		t.Error("want Engine, got nil")
	}

	// now the engine should error
	red1, err = reader_testing.New(log, deadServer.URL, "c", "ddc", time.Hour, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	e, err = expipe.New(ctx, log, rec, map[string]reader.DataReader{red1.Name(): red1, red2.Name(): red2})
	if err == nil {
		t.Error("want ErrPing, got nil")
	}

	if _, ok := errors.Cause(err).(expipe.ErrPing); !ok {
		t.Errorf("want ErrPing, got (%v)", err)
	}
	if e != nil {
		t.Errorf("want (nil), got (%v)", e)
	}
}

func TestEngineShutsDownOnAllReadersGoOutOfScope(t *testing.T) {
	t.Parallel()
	stopReader1 := uint32(0)
	stopReader2 := uint32(0)
	readerInterval := time.Millisecond * 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	red1, err := reader_testing.New(log, testServer.URL, "reader1_example", "example_type", readerInterval, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	red2, err := reader_testing.New(log, testServer.URL, "reader2_example", "example_type", readerInterval, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	red1.ReadFunc = func(job *token.Context) (*reader.Result, error) {
		if atomic.LoadUint32(&stopReader1) > 0 {
			return nil, reader.ErrBackoffExceeded
		}
		resp := &reader.Result{
			ID:       token.NewUID(),
			Content:  []byte(`{"devil":666}`),
			TypeName: red1.TypeName(),
			Mapper:   red1.Mapper(),
		}
		return resp, nil
	}

	red2.ReadFunc = func(job *token.Context) (*reader.Result, error) {
		if atomic.LoadUint32(&stopReader2) > 0 {
			return nil, reader.ErrBackoffExceeded
		}
		resp := &reader.Result{
			ID:       token.NewUID(),
			Content:  []byte(`{"devil":666}`),
			TypeName: red2.TypeName(),
			Mapper:   red2.Mapper(),
		}
		return resp, nil
	}

	rec, err := recorder_testing.New(ctx, log, "recorder_example", testServer.URL, "intexName", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	rec.RecordFunc = func(ctx context.Context, job *recorder.Job) error { return nil }

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red1.Name(): red1, red2.Name(): red2})
	if err != nil {
		t.Fatal(err)
	}

	cleanExit := make(chan struct{})
	go func() {
		e.Start()
		cleanExit <- struct{}{}
	}()

	// check the engine is working correctly with one reader
	time.Sleep(readerInterval * 3) // making sure it reads at least once
	atomic.StoreUint32(&stopReader1, uint32(1))
	time.Sleep(readerInterval * 2) // making sure the engine is not falling over

	select {
	case <-cleanExit:
		t.Fatal("expected the engine continue")
	case <-time.After(readerInterval * 2):
	}

	time.Sleep(readerInterval * 2)
	atomic.StoreUint32(&stopReader2, uint32(1))

	select {
	case <-cleanExit:
	case <-time.After(5 * time.Second):
		t.Error("expected the engine to quit")
	}
}

func TestEngineShutsDownOnRecorderGoOutOfScope(t *testing.T) {
	t.Parallel()
	stopRecorder := uint32(0)
	readerInterval := time.Millisecond * 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	red, err := reader_testing.New(log, testServer.URL, "reader_example", "example_type", time.Millisecond*50, time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}

	red.ReadFunc = func(job *token.Context) (*reader.Result, error) {
		resp := &reader.Result{
			ID:       token.NewUID(),
			Content:  []byte(`{"devil":666}`),
			TypeName: red.TypeName(),
			Mapper:   red.Mapper(),
		}
		return resp, nil
	}

	rec, err := recorder_testing.New(ctx, log, "recorder_example", testServer.URL, "intexName", time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	rec.RecordFunc = func(ctx context.Context, job *recorder.Job) error {
		if atomic.LoadUint32(&stopRecorder) > 0 {
			return recorder.ErrBackoffExceeded
		}
		return nil
	}

	e, err := expipe.New(ctx, log, rec, map[string]reader.DataReader{red.Name(): red})
	if err != nil {
		t.Fatal(err)
	}

	cleanExit := make(chan struct{})
	go func() {
		e.Start()
		cleanExit <- struct{}{}
	}()

	// check the engine is working correctly with one reader
	time.Sleep(readerInterval * 3) // making sure it reads at least once
	atomic.StoreUint32(&stopRecorder, 1)
	time.Sleep(readerInterval * 2) // making sure the engine is not falling over

	select {
	case <-cleanExit:
	case <-time.After(5 * time.Second):
		t.Error("expected the engine to quit")
	}
}

func TestEngineWithConfigFailsOnNilReaderConf(t *testing.T) {
	ctx := context.Background()

	rec, err := recorder_testing.NewConfig("recorder_example", log, "nowhere", time.Hour, 5, "index")
	if err != nil {
		t.Fatal(err)
	}

	e, err := expipe.WithConfig(ctx, log, rec)
	if errors.Cause(err) != expipe.ErrNoReader {
		t.Errorf("want ErrNoReader, got (%v)", err)
	}
	if e != nil {
		t.Errorf("want nil, got (%v)", e)
	}

	e, err = expipe.WithConfig(ctx, log, rec, nil)
	if errors.Cause(err) != expipe.ErrNoReader {
		t.Errorf("want ErrNoReader, got (%v)", err)
	}
	if e != nil {
		t.Errorf("want nil, got (%v)", e)
	}
}
