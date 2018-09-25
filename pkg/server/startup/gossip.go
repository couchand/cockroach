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
	"google.golang.org/grpc"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/gossip"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/rpc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/metric"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitGossip(
	ambientCtx log.AmbientContext,
	nodeIDContainer *base.NodeIDContainer,
	rpcContext *rpc.Context,
	grpcServer *grpc.Server,
	stopper *stop.Stopper,
	registry *metric.Registry,
	locality roachpb.Locality,
) *gossip.Gossip {
	g := gossip.New(
		ambientCtx,
		&rpcContext.ClusterID,
		nodeIDContainer,
		rpcContext,
		grpcServer,
		stopper,
		registry,
		locality,
	)

	return g
}
