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
	"github.com/cockroachdb/cockroach/pkg/settings/cluster"
	"github.com/cockroachdb/cockroach/pkg/sql"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/metric"
	"github.com/cockroachdb/cockroach/pkg/util/mon"
)

func InitPGServer(
	ambientCtx log.AmbientContext,
	cfg *base.Config,
	st *cluster.Settings,
	memMetrics sql.MemoryMetrics,
	memMonitor *mon.BytesMonitor,
	histogramWindowInterval time.Duration,
	execCfg *sql.ExecutorConfig,
	registry *metric.Registry,
) *pgwire.Server {
	pgServer := pgwire.MakeServer(
		ambientCtx,
		cfg,
		st,
		memMetrics,
		memMonitor,
		histogramWindowInterval,
		execCfg,
	)

	registry.AddMetricStruct(pgServer.Metrics())
	registry.AddMetricStruct(pgServer.StatementCounters())
	registry.AddMetricStruct(pgServer.EngineMetrics())

	return pgServer
}
