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
	"github.com/cockroachdb/cockroach/pkg/util/metric"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitTLSConfigs(
	cfg *base.Config,
	registry *metric.Registry,
	stopper *stop.Stopper,
) error {
	// Attempt to load TLS configs right away, failures are permanent.
	if certMgr, err := cfg.InitializeNodeTLSConfigs(stopper); err != nil {
		return err
	} else if certMgr != nil {
		// The certificate manager is non-nil in secure mode.
		registry.AddMetricStruct(certMgr.Metrics())
	}

	return nil
}
