// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package datatype

import (
    "io"

    "github.com/antonholmquist/jason"
    "github.com/arsham/expvastic/lib"
)

// JobResultDataTypes generates a list of DataType and puts them inside the DataContainer
// TODO: bypass the operation that won't be converted in any way. They are not supposed to be read
// and converted back.
// TODO: this operation can happen only once. Lazy load the thing
func JobResultDataTypes(r io.Reader) DataContainer {
    obj, err := jason.NewObjectFromReader(r)
    if err != nil {
        return &Container{Err: err}
    }
    return getJasonValues("", obj.Map())
}

// FromJason returns an instance of DataType from jason value
func FromJason(key string, value jason.Value) (DataType, error) {
    var (
        err error
        s   string
        f   float64
    )
    if s, err = value.String(); err == nil {
        return &StringType{key, s}, nil
    } else if f, err = value.Float64(); err == nil {
        return &FloatType{key, f}, nil
    }
    return nil, ErrUnidentifiedJason
}

func floatListValues(key string, values []*jason.Value) []DataType {
    if len(values) == 0 {
        // empty list
        return []DataType{&FloatListType{key, []float64{}}}
    }

    if lib.IsGCType(key) {
        return gcListValues(key, values)
    }

    if _, err := values[0].Float64(); err == nil {
        res := make([]float64, len(values))
        for i, val := range values {
            if r, err := val.Float64(); err == nil {
                res[i] = r
            }
        }
        return []DataType{&FloatListType{key, res}}
    }
    return nil
}

func gcListValues(key string, values []*jason.Value) (result []DataType) {
    if _, err := values[0].Float64(); err == nil {
        res := make([]uint64, len(values))
        for i, val := range values {
            if r, err := val.Float64(); err == nil {
                res[i] = uint64(r)
            }
        }
        result = []DataType{&GCListType{key, res}}
    }
    return
}
