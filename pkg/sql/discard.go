// Copyright 2015 The Cockroach Authors.
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
	"golang.org/x/net/context"

	"github.com/cockroachdb/cockroach/pkg/sql/parser"
)

// Resets the session to the initial state.  According to the PostgreSQL docs
// (https://www.postgresql.org/docs/9.6/static/sql-discard.html), this is equivalent
// to the following sequence (supported functionality marked):
//
//     SET SESSION AUTHORIZATION DEFAULT;
//   * RESET ALL;
//   * DEALLOCATE ALL;
//     CLOSE ALL;
//     UNLISTEN *;
//     SELECT pg_advisory_unlock_all();
//     DISCARD PLANS;
//     DISCARD SEQUENCES;
//     DISCARD TEMP;
//
//   SET SESSION AUTHORIZATION DEFAULT;
// 	   Resets the session and current users to the originally-authenticated user.
//     As there is no mechanism for changing these, this is unneeded.
//
//   RESET ALL;
//     Resets all session settings to their default values.
//     Supported.
//
//   DEALLOCATE ALL;
//     Clears all prepared statements.
//     Supported.
//
//   CLOSE ALL;
//     Closes all currently-open cursors.
//     As there is no support for cursors, this is unneeded.
//
//   UNLISTEN *;
//     Removes all subscriptions for notifications.
//     As there is no support for notifications, this is unneeded.
//
//   SELECT pg_advisory_unlock_all();
//     Clears all current advisory locks.
//     As there is no support for advisory locks, this is uneeded.
//
//   DISCARD PLANS;
//     Discards all cached query plans.
//     As there is no caching of query plans, this is unneeded.
//
//   DISCARD SEQUENCES;
//     Discards all cached sequence data.
//     As there is no support for sequences, this is unneeded.
//
//   DISCARD TEMP;
//     Drops all temporary tables.
//     As there is no support for temporary tables, this is unneeded.
//
// Privileges: None.
//
// TODO(couchand): I think maybe this code should actually live in executor.go +1393.
//   This would help with two things:
//   - We could use the implicitTxn flag to ensure that we're not in a transaction.
//   - We wouldn't need to pollute the query planner with an empty node.
//   There is one main reason we can't do that immediately: the sessionVar struct
//   method Reset takes a planner as input.  It looks like the current var implementations
//   never use anything from the planner except session, so that interface could be
//   changed to take the session, allowing this code to be moved to the executor.
//   That would, unfortunately, make the interface for Reset look different than the
//   ones for Set and Get.
func (p *planner) Discard(ctx context.Context, n *parser.Discard) (planNode, error) {

	switch n.Mode {
	case parser.DiscardModeAll:
		// TODO(couchand): Check that we're not in a transaction.

		// RESET ALL
		for _, v := range varGen {
			if v.Reset != nil {
				if err := v.Reset(p); err != nil {
					return nil, err
				}
			}
		}

		// DEALLOCATE ALL
		p.session.PreparedStatements.DeleteAll(ctx)
	}

	return &emptyNode{}, nil
}
