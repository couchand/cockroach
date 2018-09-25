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
	"context"
	"time"

	"google.golang.org/grpc"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/gossip"
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/jobs"
	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/rpc"
	"github.com/cockroachdb/cockroach/pkg/rpc/nodedialer"
	"github.com/cockroachdb/cockroach/pkg/settings/cluster"
	"github.com/cockroachdb/cockroach/pkg/sql"
	"github.com/cockroachdb/cockroach/pkg/sql/distsqlrun"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire"
	"github.com/cockroachdb/cockroach/pkg/sql/sessiondata"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlutil"
	"github.com/cockroachdb/cockroach/pkg/storage/engine"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/metric"
	"github.com/cockroachdb/cockroach/pkg/util/mon"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitDistSQLServer(
	ctx context.Context,
	ambientCtx log.AmbientContext,
	st *cluster.Settings,
	db *client.DB,
	internalExecutor *sql.InternalExecutor,
	tcsFactory *kv.TxnCoordSenderFactory,
	clock *hlc.Clock,
	rpcContext *rpc.Context,
	stopper *stop.Stopper,
	nodeIDContainer *base.NodeIDContainer,
	tempEngine engine.MapProvidingEngine,
	tempStorageConfig base.TempStorageConfig,
	memMonitor *mon.BytesMonitor,
	histogramWindowInterval time.Duration,
	jobRegistry *jobs.Registry,
	gossip *gossip.Gossip,
	nodeDialer *nodedialer.Dialer,
	leaseMgr *sql.LeaseManager,
	testingKnobs base.TestingKnobs,
	grpcServer *grpc.Server,
	registry *metric.Registry,
) *distsqlrun.ServerImpl {
	distSQLMetrics := distsqlrun.MakeDistSQLMetrics(histogramWindowInterval)
	registry.AddMetricStruct(distSQLMetrics)

	// Set up the DistSQL server.
	distSQLCfg := distsqlrun.ServerConfig{
		AmbientContext: ambientCtx,
		Settings:       st,
		DB:             db,
		Executor:       internalExecutor,
		FlowDB:         client.NewDB(ambientCtx, tcsFactory, clock),
		RPCContext:     rpcContext,
		Stopper:        stopper,
		NodeID:         nodeIDContainer,
		ClusterID:      &rpcContext.ClusterID,

		TempStorage: tempEngine,
		DiskMonitor: tempStorageConfig.Mon,

		ParentMemoryMonitor: memMonitor,

		Metrics: &distSQLMetrics,

		JobRegistry:  jobRegistry,
		Gossip:       gossip,
		NodeDialer:   nodeDialer,
		LeaseManager: leaseMgr,
	}
	if distSQLTestingKnobs := testingKnobs.DistSQL; distSQLTestingKnobs != nil {
		distSQLCfg.TestingKnobs = *distSQLTestingKnobs.(*distsqlrun.TestingKnobs)
	}

	distSQLServer := distsqlrun.NewServer(ctx, distSQLCfg)
	distsqlrun.RegisterDistSQLServer(grpcServer, distSQLServer)

	return distSQLServer
}

func FinalizeDistSQLServer(
	distSQLServer *distsqlrun.ServerImpl,
	pgServer *pgwire.Server,
	sqlMemMetrics sql.MemoryMetrics,
	st *cluster.Settings,
) {
	// Now that we have a pgwire.Server (which has a sql.Server), we can close a
	// circular dependency between the distsqlrun.Server and sql.Server and set
	// SessionBoundInternalExecutorFactory.
	distSQLServer.ServerConfig.SessionBoundInternalExecutorFactory =
		func(
			ctx context.Context, sessionData *sessiondata.SessionData,
		) sqlutil.InternalExecutor {
			ie := sql.MakeSessionBoundInternalExecutor(
				ctx,
				sessionData,
				pgServer.SQLServer,
				sqlMemMetrics,
				st,
			)
			return &ie
		}
}
