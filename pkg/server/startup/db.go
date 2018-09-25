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
	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitDB(
	ambientCtx log.AmbientContext,
	tcsFactory *kv.TxnCoordSenderFactory,
	clock *hlc.Clock,
	nodeIDContainer *base.NodeIDContainer,
	stopper *stop.Stopper,
) *client.DB {
	dbCtx := client.DefaultDBContext()
	dbCtx.NodeID = nodeIDContainer
	dbCtx.Stopper = stopper

	db := client.NewDBWithContext(ambientCtx, tcsFactory, clock, dbCtx)

	return db
}
