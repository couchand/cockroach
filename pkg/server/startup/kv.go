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
	"time"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/gossip"
	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/rpc"
	"github.com/cockroachdb/cockroach/pkg/rpc/nodedialer"
	"github.com/cockroachdb/cockroach/pkg/settings/cluster"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/metric"
	"github.com/cockroachdb/cockroach/pkg/util/retry"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitKV(
	testingKnobs base.TestingKnobs,
	ambientCtx log.AmbientContext,
	st *cluster.Settings,
	clock *hlc.Clock,
	rpcContext *rpc.Context,
	retryOptions retry.Options,
	stopper *stop.Stopper,
	nodeDialer *nodedialer.Dialer,
	registry *metric.Registry,
	gossip *gossip.Gossip,
	histogramWindowInterval time.Duration,
	linearizable bool,
) (
	distSender *kv.DistSender,
	tcsFactory *kv.TxnCoordSenderFactory,
	txnMetrics kv.TxnMetrics,
) {
	var clientTestingKnobs kv.ClientTestingKnobs
	if kvKnobs := testingKnobs.KVClient; kvKnobs != nil {
		clientTestingKnobs = *kvKnobs.(*kv.ClientTestingKnobs)
	}

	distSender = InitDistSender(
		ambientCtx,
		st,
		clock,
		rpcContext,
		retryOptions,
		clientTestingKnobs,
		stopper,
		nodeDialer,
		registry,
		gossip,
	)

	txnMetrics = kv.MakeTxnMetrics(histogramWindowInterval)
	registry.AddMetricStruct(txnMetrics)

	tcsFactory = InitTxnCoordSenderFactory(
		ambientCtx,
		st,
		clock,
		stopper,
		linearizable,
		txnMetrics,
		clientTestingKnobs,
		registry,
		distSender,
	)

	return
}
