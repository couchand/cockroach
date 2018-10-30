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
	"fmt"
	"time"

	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
)

// StmtExecution collects statistics related to the execution of a single
// statement within a transaction.
type StmtExecution struct {
	EndTime     time.Time
	Stmt        string
	DistSQLUsed bool
	OptUsed     bool
	RetryCount  int64
	NumRows     int64
	Err         error
	ParseLat    float64
	PlanLat     float64
	RunLat      float64
	ServiceLat  float64
	OverheadLat float64
}

type TxnAttempt struct {
	Statements []StmtExecution
}

// TxnExecution collects statistics related to a transaction.
type TxnExecution struct {
	Duration float64
	Aborted  bool
	Attempts []TxnAttempt
}

// txnStatsCollector collects statistics related to a transaction.
type txnStatsCollector struct {
	attempts []TxnAttempt
}

func newTxnStatsCollector() *txnStatsCollector {
	attempts := []TxnAttempt{
		TxnAttempt{
			Statements: make([]StmtExecution, 0, 10),
		},
	}
	return &txnStatsCollector{
		attempts: attempts,
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
	/*
		if ts.startTime == (time.Time{}) {
			log.Fatalf(context.Background(), "attempted to record a statement outside a transaction!")
		}
	*/
	endTime := timeutil.Now()
	stmtEx := StmtExecution{
		EndTime:     endTime,
		Stmt:        anonymizeStmt(stmt),
		DistSQLUsed: distSQLUsed,
		OptUsed:     optUsed,
		RetryCount:  int64(retryCount),
		NumRows:     int64(numRows),
		Err:         err,
		ParseLat:    parseLat,
		PlanLat:     planLat,
		RunLat:      runLat,
		ServiceLat:  serviceLat,
		OverheadLat: overheadLat,
	}

	log.Shout(context.Background(), log.Severity_ERROR, fmt.Sprintf("recording statement: %s",
		stmtEx))

	curAttempt := &ts.attempts[len(ts.attempts)-1]
	curAttempt.Statements = append(curAttempt.Statements, stmtEx)
}

func (ts *txnStatsCollector) reset() {
	ts.attempts = ts.attempts[:1]
	ts.attempts[0].Statements = ts.attempts[0].Statements[:0]
}

func (ts *txnStatsCollector) start() {
}

func (ts *txnStatsCollector) restart() {
	ts.attempts = append(ts.attempts, TxnAttempt{
		Statements: make([]StmtExecution, 0, 10),
	})
}

func (ts *txnStatsCollector) record(aborted bool) TxnExecution {
	if len(ts.attempts[0].Statements) == 0 {
		return TxnExecution{}
	}

	firstStmt := ts.attempts[0].Statements[0]
	postLat := timeutil.Since(firstStmt.EndTime).Seconds()
	duration := postLat + firstStmt.ServiceLat
	return TxnExecution{
		Duration: duration,
		Attempts: ts.attempts,
		Aborted:  aborted,
	}
}

func (ts *txnStatsCollector) commit() TxnExecution {
	return ts.record(false /* aborted */)
}

func (ts *txnStatsCollector) abort() TxnExecution {
	return ts.record(true /* aborted */)
}

func (ts *txnStatsCollector) acceptAdvanceInfo(advInfo advanceInfo) []TxnExecution {
	switch advInfo.txnEvent {
	case txnStart:
		ts.start()
	case txnRestart:
		ts.restart()
	case txnCommit:
		return []TxnExecution{ts.commit()}
	case txnAborted:
		return []TxnExecution{ts.abort()}
	}

	return nil
}
