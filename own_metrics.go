// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package expvastic

import "expvar"

var (
    expRecorders = expvar.NewInt("Recorders")
    expReaders   = expvar.NewInt("Readers")
)
