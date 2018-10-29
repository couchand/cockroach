// Copyright 2016 The Cockroach Authors.
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

package sql

import (
	"context"
	"time"

	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
)

// stmtExecution collects statistics related to the execution of a single
// statement within a transaction.
type stmtExecution struct {
	// The start time offset relative to the transaction start.
	startTime   float64
	stmt        Statement
	distSQLUsed bool
	optUsed     bool
	retryCount  int
	numRows     int
	err         error
	parseLat    float64
	planLat     float64
	runLat      float64
	serviceLat  float64
	overheadLat float64
}

// txnExecution collects statistics related to a transaction.
type txnExecution struct {
	startTime  time.Time
	duration   float64
	statements []stmtExecution
}

// txnStatsCollector collects statistics related to a transaction.
type txnStatsCollector struct {
	startTime  time.Time
	statements []stmtExecution
}

func newTxnStatsCollector() *txnStatsCollector {
	return &txnStatsCollector{
		startTime:  time.Time{},
		statements: make([]stmtExecution, 10),
	}
}

func (ts *txnStatsCollector) recordStatement(
	stmt Statement,
	distSQLUsed bool,
	optUsed bool,
	retryCount int,
	numRows int,
	err error,
	parseLat, planLat, runLat, serviceLat, overheadLat float64,
) {
	if ts.startTime == (time.Time{}) {
		log.Fatalf(context.Background(), "attempted to record a statement outside a transaction!")
	}

	startTime := timeutil.Since(ts.startTime).Seconds()
	ts.statements = append(ts.statements, stmtExecution{
		startTime:   startTime,
		stmt:        stmt,
		distSQLUsed: distSQLUsed,
		optUsed:     optUsed,
		retryCount:  retryCount,
		numRows:     numRows,
		err:         err,
		parseLat:    parseLat,
		planLat:     planLat,
		runLat:      runLat,
		serviceLat:  serviceLat,
		overheadLat: overheadLat,
	})
}

func (ts *txnStatsCollector) reset() {
	ts.startTime = time.Time{}
	ts.statements = ts.statements[:0]
}

func (ts *txnStatsCollector) start() {
	ts.startTime = timeutil.Now()
}

func (ts *txnStatsCollector) export() txnExecution {
	return txnExecution{
		startTime:  ts.startTime,
		duration:   timeutil.Since(ts.startTime).Seconds(),
		statements: ts.statements,
	}
}
