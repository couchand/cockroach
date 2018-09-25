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
	"github.com/cockroachdb/cockroach/pkg/gossip"
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/sql"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitLeaseManager(
	ambientCtx log.AmbientContext,
	stopper *stop.Stopper,
	leaseManagerConfig *base.LeaseManagerConfig,
	testingKnobs base.TestingKnobs,
) *sql.LeaseManager {
	var lmKnobs sql.LeaseManagerTestingKnobs
	if leaseManagerTestingKnobs := testingKnobs.SQLLeaseManager; leaseManagerTestingKnobs != nil {
		lmKnobs = *leaseManagerTestingKnobs.(*sql.LeaseManagerTestingKnobs)
	}

	leaseMgr := sql.NewLeaseManager(
		ambientCtx,
		nil, /* execCfg - will be set later because of circular dependencies */
		lmKnobs,
		stopper,
		leaseManagerConfig,
	)

	return leaseMgr
}

func FinalizeLeaseManager(
	leaseMgr *sql.LeaseManager,
	execCfg *sql.ExecutorConfig,
	stopper *stop.Stopper,
	db *client.DB,
	gossip *gossip.Gossip,
) {
	leaseMgr.SetExecCfg(execCfg)
	leaseMgr.RefreshLeases(stopper, db, gossip)
	leaseMgr.PeriodicallyRefreshSomeLeases()
}
