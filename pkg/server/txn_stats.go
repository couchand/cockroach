// Copyright 2014 The Cockroach Authors.
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

package server

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/server/serverpb"
	"github.com/cockroachdb/cockroach/pkg/util/log"
)

func (s *statusServer) Transactions(
	ctx context.Context, req *serverpb.TransactionsRequest,
) (*serverpb.TransactionsResponse, error) {
	ctx = propagateGatewayMetadata(ctx)
	ctx = s.AnnotateCtx(ctx)

	response := &serverpb.TransactionsResponse{
		Transactions: []serverpb.TxnExecution{},
	}

	localReq := &serverpb.TransactionsRequest{
		NodeID: "local",
	}

	if len(req.NodeID) > 0 {
		requestedNodeID, local, err := s.parseNodeID(req.NodeID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if local {
			return s.TransactionsLocal(ctx)
		}
		status, err := s.dialNode(ctx, requestedNodeID)
		if err != nil {
			return nil, err
		}
		return status.Transactions(ctx, localReq)
	}

	dialFn := func(ctx context.Context, nodeID roachpb.NodeID) (interface{}, error) {
		client, err := s.dialNode(ctx, nodeID)
		return client, err
	}
	nodeTransactions := func(ctx context.Context, client interface{}, _ roachpb.NodeID) (interface{}, error) {
		status := client.(serverpb.StatusClient)
		return status.Transactions(ctx, localReq)
	}

	if err := s.iterateNodes(ctx, fmt.Sprintf("transaction statistics for node %s", req.NodeID),
		dialFn,
		nodeTransactions,
		func(nodeID roachpb.NodeID, resp interface{}) {
			transactionsResp := resp.(*serverpb.TransactionsResponse)
			response.Transactions = append(response.Transactions, transactionsResp.Transactions...)
		},
		func(nodeID roachpb.NodeID, err error) {
			// TODO(couchand): do something here...
		},
	); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *statusServer) TransactionsLocal(ctx context.Context) (*serverpb.TransactionsResponse, error) {
	txnStats := s.admin.server.pgServer.SQLServer.TxnStats

	resp := &serverpb.TransactionsResponse{
		Transactions: make([]serverpb.TxnExecution, len(txnStats)),
	}

	for i, txn := range txnStats {
		resp.Transactions[i] = serverpb.TxnExecution{
			Duration: txn.Duration,
			Aborted:  txn.Aborted,
			Attempts: make([]serverpb.TxnAttempt, len(txn.Attempts)),
		}

		for j, attempt := range txn.Attempts {
			resp.Transactions[i].Attempts[j] = serverpb.TxnAttempt{
				Statements: make([]serverpb.StmtExecution, len(attempt.Statements)),
			}

			for k, stmt := range attempt.Statements {
				log.Shout(ctx, log.Severity_ERROR, fmt.Sprintf("serializing stmt: %s", stmt))

				var err string
				if stmt.Err != nil {
					err = stmt.Err.Error()
				}

				resp.Transactions[i].Attempts[j].Statements[k] = serverpb.StmtExecution{
					EndTime:     stmt.EndTime,
					Stmt:        stmt.Stmt,
					DistSQLUsed: stmt.DistSQLUsed,
					OptUsed:     stmt.OptUsed,
					RetryCount:  stmt.RetryCount,
					NumRows:     stmt.NumRows,
					Err:         err,
					ParseLat:    stmt.ParseLat,
					PlanLat:     stmt.PlanLat,
					RunLat:      stmt.RunLat,
					ServiceLat:  stmt.ServiceLat,
					OverheadLat: stmt.OverheadLat,
				}
			}
		}
	}

	return resp, nil
}
