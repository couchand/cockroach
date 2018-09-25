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
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/settings/cluster"
	"github.com/cockroachdb/cockroach/pkg/storage"
	"github.com/cockroachdb/cockroach/pkg/storage/engine"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/metric"
)

func InitNodeLiveness(
	ambientCtx log.AmbientContext,
	cfg base.RaftConfig,
	clock *hlc.Clock,
	db *client.DB,
	engines []engine.Engine,
	gossip *gossip.Gossip,
	st *cluster.Settings,
	histogramWindowInterval time.Duration,
	registry *metric.Registry,
) *storage.NodeLiveness {
	nlActive, nlRenewal := cfg.NodeLivenessDurations()

	nodeLiveness := storage.NewNodeLiveness(
		ambientCtx,
		clock,
		db,
		engines,
		gossip,
		nlActive,
		nlRenewal,
		st,
		histogramWindowInterval,
	)
	registry.AddMetricStruct(nodeLiveness.Metrics())

	return nodeLiveness
}
