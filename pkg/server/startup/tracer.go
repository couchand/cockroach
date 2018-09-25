// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package startup

import (
	"github.com/pkg/errors"

	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitTracer(ctx log.AmbientContext, stopper *stop.Stopper) {
	if ctx.Tracer == nil {
		panic(errors.New("no tracer set in AmbientCtx"))
	}

	// If the tracer has a Close function, call it after the server stops.
	if tr, ok := ctx.Tracer.(stop.Closer); ok {
		stopper.AddCloser(tr)
	}
}
