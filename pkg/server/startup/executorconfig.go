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

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/gossip"
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/jobs"
	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/rpc"
	"github.com/cockroachdb/cockroach/pkg/rpc/nodedialer"
	"github.com/cockroachdb/cockroach/pkg/server/serverpb"
	"github.com/cockroachdb/cockroach/pkg/server/status"
	"github.com/cockroachdb/cockroach/pkg/settings/cluster"
	"github.com/cockroachdb/cockroach/pkg/sql"
	"github.com/cockroachdb/cockroach/pkg/sql/distsqlrun"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/stats"
	"github.com/cockroachdb/cockroach/pkg/storage"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitExecutorConfig(
	st *cluster.Settings,
	ambientCtx log.AmbientContext,
	db *client.DB,
	gossip *gossip.Gossip,
	recorder *status.MetricsRecorder,
	distSender *kv.DistSender,
	rpcContext *rpc.Context,
	leaseMgr *sql.LeaseManager,
	clock *hlc.Clock,
	distSQLServer *distsqlrun.ServerImpl,
	statusServer serverpb.StatusServer,
	sessionRegistry *sql.SessionRegistry,
	jobRegistry *jobs.Registry,
	histogramWindowInterval time.Duration,
	stopper *stop.Stopper,
	nodeLiveness *storage.NodeLiveness,
	nodeDialer *nodedialer.Dialer,
	sqlTableStatCacheSize int,
	internalExecutor *sql.InternalExecutor,
	sqlAuditLogDirName *log.DirName,
	connResultsBufferBytes int,
	testingKnobs base.TestingKnobs,
	nodeInfo sql.NodeInfo,
) sql.ExecutorConfig {
	ctx := ambientCtx.AnnotateCtx(context.Background())

	virtualSchemas, err := sql.NewVirtualSchemaHolder(ctx, st)
	if err != nil {
		log.Fatal(ctx, err)
	}

	var sqlExecutorTestingKnobs *sql.ExecutorTestingKnobs
	if k := testingKnobs.SQLExecutor; k != nil {
		sqlExecutorTestingKnobs = k.(*sql.ExecutorTestingKnobs)
	} else {
		sqlExecutorTestingKnobs = new(sql.ExecutorTestingKnobs)
	}

	execCfg := sql.ExecutorConfig{
		Settings:                st,
		NodeInfo:                nodeInfo,
		AmbientCtx:              ambientCtx,
		DB:                      db,
		Gossip:                  gossip,
		MetricsRecorder:         recorder,
		DistSender:              distSender,
		RPCContext:              rpcContext,
		LeaseManager:            leaseMgr,
		Clock:                   clock,
		DistSQLSrv:              distSQLServer,
		StatusServer:            statusServer,
		SessionRegistry:         sessionRegistry,
		JobRegistry:             jobRegistry,
		VirtualSchemas:          virtualSchemas,
		HistogramWindowInterval: histogramWindowInterval,
		RangeDescriptorCache:    distSender.RangeDescriptorCache(),
		LeaseHolderCache:        distSender.LeaseHolderCache(),
		TestingKnobs:            sqlExecutorTestingKnobs,

		DistSQLPlanner: sql.NewDistSQLPlanner(
			ctx,
			distsqlrun.Version,
			st,
			// The node descriptor will be set later, once it is initialized.
			roachpb.NodeDescriptor{},
			rpcContext,
			distSQLServer,
			distSender,
			gossip,
			stopper,
			nodeLiveness,
			sqlExecutorTestingKnobs.DistSQLPlannerKnobs,
			nodeDialer,
		),

		TableStatsCache: stats.NewTableStatisticsCache(
			sqlTableStatCacheSize,
			gossip,
			db,
			internalExecutor,
		),

		ExecLogger: log.NewSecondaryLogger(
			nil /* dirName */, "sql-exec", true /* enableGc */, false, /*forceSyncWrites*/
		),

		AuditLogger: log.NewSecondaryLogger(
			sqlAuditLogDirName, "sql-audit", true /*enableGc*/, true, /*forceSyncWrites*/
		),

		ConnResultsBufferBytes: connResultsBufferBytes,
	}

	if sqlSchemaChangerTestingKnobs := testingKnobs.SQLSchemaChanger; sqlSchemaChangerTestingKnobs != nil {
		execCfg.SchemaChangerTestingKnobs = sqlSchemaChangerTestingKnobs.(*sql.SchemaChangerTestingKnobs)
	} else {
		execCfg.SchemaChangerTestingKnobs = new(sql.SchemaChangerTestingKnobs)
	}
	if distSQLRunTestingKnobs := testingKnobs.DistSQL; distSQLRunTestingKnobs != nil {
		execCfg.DistSQLRunTestingKnobs = distSQLRunTestingKnobs.(*distsqlrun.TestingKnobs)
	} else {
		execCfg.DistSQLRunTestingKnobs = new(distsqlrun.TestingKnobs)
	}
	if sqlEvalContext := testingKnobs.SQLEvalContext; sqlEvalContext != nil {
		execCfg.EvalContextTestingKnobs = *sqlEvalContext.(*tree.EvalContextTestingKnobs)
	}

	return execCfg
}

func FinalizeExecutorConfig(
	execCfg *sql.ExecutorConfig,
	internalExecutor *sql.InternalExecutor,
) {
	execCfg.InternalExecutor = internalExecutor
}
