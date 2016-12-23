// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

// Package datatype contains necessary logic to sanitise a JSON object coming from a reader. This
// package is subjected to change.
package datatype

import (
    "errors"
    "expvar"
    "fmt"
    "strings"
)

var (
    stringTypeCount = expvar.NewInt("StringType Count")
    floatTypeCount  = expvar.NewInt("FloatType Count")
    gcListTypeCount = expvar.NewInt("GCListType Count")
    byteTypeCount   = expvar.NewInt("ByteType Count")
)

const (
    // BYTE ..
    BYTE = 1.0
    // KILOBYTE ..
    KILOBYTE = 1024 * BYTE
    // MEGABYTE ..
    MEGABYTE = 1024 * KILOBYTE
)

// TODO: refactor to use byte slices

// ErrUnidentifiedJason .
var ErrUnidentifiedJason = errors.New("unidentified jason value")

// DataType implements Stringer and Marshal/Unmarshal
type DataType interface {
    fmt.Stringer
}

// FloatType represents a pair of key values that the value is a float64
type FloatType struct {
    Key   string
    Value float64
}

// String satisfies the Stringer interface
func (f FloatType) String() string {
    floatTypeCount.Add(1)
    return fmt.Sprintf(`"%s":%f`, f.Key, f.Value)
}

// StringType represents a pair of key values that the value is a string
type StringType struct {
    Key   string
    Value string
}

// String satisfies the Stringer interface
func (s StringType) String() string {
    stringTypeCount.Add(1)
    return fmt.Sprintf(`"%s":"%s"`, s.Key, s.Value)
}

// FloatListType represents a pair of key values that the value is a list of floats
type FloatListType struct {
    Key   string
    Value []float64
}

// String satisfies the Stringer interface
func (fl FloatListType) String() string {
    floatTypeCount.Add(1)
    list := make([]string, len(fl.Value))
    for i, v := range fl.Value {
        list[i] = fmt.Sprintf("%f", v)
    }
    return fmt.Sprintf(`"%s":[%s]`, fl.Key, strings.Join(list, ","))
}

// GCListType represents a pair of key values of GC list info
type GCListType struct {
    Key   string
    Value []uint64
}

// String satisfies the Stringer interface
func (flt GCListType) String() string {
    // We are filtering, therefore we don't know the size
    var list []string
    for _, v := range flt.Value {
        if v > 0 {
            list = append(list, fmt.Sprintf("%d", v/1000))
        }
    }
    gcListTypeCount.Add(1)
    return fmt.Sprintf(`"%s":[%s]`, flt.Key, strings.Join(list, ","))
}

// ByteType represents a pair of key values in which the value represents bytes
// It converts the value to MB
type ByteType struct {
    Key   string
    Value float64
}

// String satisfies the Stringer interface
func (b ByteType) String() string {
    byteTypeCount.Add(1)
    return fmt.Sprintf(`"%s":%f`, b.Key, b.Value/MEGABYTE)
}
