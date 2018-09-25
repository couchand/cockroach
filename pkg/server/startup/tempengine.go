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
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/storage/engine"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/stop"
)

func InitTempEngine(
	useStoreSpec base.StoreSpec,
	tempStorageConfig base.TempStorageConfig,
	stopper *stop.Stopper,
) (engine.MapProvidingEngine, error) {
	tempEngine, err := engine.NewTempEngine(tempStorageConfig, useStoreSpec)
	if err != nil {
		return nil, errors.Wrap(err, "could not create temp storage")
	}
	stopper.AddCloser(tempEngine)
	// Remove temporary directory linked to tempEngine after closing
	// tempEngine.
	stopper.AddCloser(stop.CloserFn(func() {
		var err error
		if useStoreSpec.InMemory {
			// First store is in-memory so we remove the temp
			// directory directly since there is no record file.
			err = os.RemoveAll(tempStorageConfig.Path)
		} else {
			// If record file exists, we invoke CleanupTempDirs to
			// also remove the record after the temp directory is
			// removed.
			recordPath := filepath.Join(useStoreSpec.Path, base.TempDirsRecordFilename)
			err = engine.CleanupTempDirs(recordPath)
		}
		if err != nil {
			log.Errorf(context.TODO(), "could not remove temporary store directory: %v", err.Error())
		}
	}))

	return tempEngine, nil
}
