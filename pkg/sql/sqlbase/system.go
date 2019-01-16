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

package sqlbase

import (
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/config"
	"github.com/cockroachdb/cockroach/pkg/keys"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/security"
	"github.com/cockroachdb/cockroach/pkg/sql/privilege"
)

func init() {
	// We use a hook to avoid a dependency on the sqlbase package. We
	// should probably move keys/protos elsewhere.
	config.SplitAtIDHook = SplitAtIDHook
}

// SplitAtIDHook determines whether a specific descriptor ID
// should be considered for a split at all. If it is a database
// or a view table descriptor, it should not be considered.
func SplitAtIDHook(id uint32, cfg *config.SystemConfig) bool {
	descVal := cfg.GetDesc(MakeDescMetadataKey(ID(id)))
	if descVal == nil {
		return false
	}
	var desc Descriptor
	if err := descVal.GetProto(&desc); err != nil {
		return false
	}
	if dbDesc := desc.GetDatabase(); dbDesc != nil {
		return false
	}
	if tableDesc := desc.GetTable(); tableDesc != nil {
		if viewStr := tableDesc.GetViewQuery(); viewStr != "" {
			return false
		}
	}
	return true
}

// sql CREATE commands and full schema for each system table.
// These strings are *not* used at runtime, but are checked by the
// `TestSystemTableLiterals` test that compares the table generated by
// evaluating the `CREATE TABLE` statement to the descriptor literal that is
// actually used at runtime.

// These system tables are part of the system config.
const (
	NamespaceTableSchema = `
CREATE TABLE system.namespace (
  "parentID" INT8,
  name       STRING,
  id         INT8,
  PRIMARY KEY ("parentID", name)
);`

	DescriptorTableSchema = `
CREATE TABLE system.descriptor (
  id         INT8 PRIMARY KEY,
  descriptor BYTES
);`

	UsersTableSchema = `
CREATE TABLE system.users (
  username         STRING PRIMARY KEY,
  "hashedPassword" BYTES,
  "isRole"         BOOL NOT NULL DEFAULT false
);`

	// Zone settings per DB/Table.
	ZonesTableSchema = `
CREATE TABLE system.zones (
  id     INT8 PRIMARY KEY,
  config BYTES
);`

	SettingsTableSchema = `
CREATE TABLE system.settings (
	name              STRING    NOT NULL PRIMARY KEY,
	value             STRING    NOT NULL,
	"lastUpdated"     TIMESTAMP NOT NULL DEFAULT now(),
	"valueType"       STRING,
	FAMILY (name, value, "lastUpdated", "valueType")
);`
)

// These system tables are not part of the system config.
const (
	LeaseTableSchema = `
CREATE TABLE system.lease (
  "descID"   INT8,
  version    INT8,
  "nodeID"   INT8,
  expiration TIMESTAMP,
  PRIMARY KEY ("descID", version, expiration, "nodeID")
);`

	EventLogTableSchema = `
CREATE TABLE system.eventlog (
  timestamp     TIMESTAMP  NOT NULL,
  "eventType"   STRING     NOT NULL,
  "targetID"    INT8       NOT NULL,
  "reportingID" INT8       NOT NULL,
  info          STRING,
  "uniqueID"    BYTES      DEFAULT uuid_v4(),
  PRIMARY KEY (timestamp, "uniqueID")
);`

	// rangelog is currently envisioned as a wide table; many different event
	// types can be recorded to the table.
	RangeEventTableSchema = `
CREATE TABLE system.rangelog (
  timestamp      TIMESTAMP  NOT NULL,
  "rangeID"      INT8       NOT NULL,
  "storeID"      INT8       NOT NULL,
  "eventType"    STRING     NOT NULL,
  "otherRangeID" INT8,
  info           STRING,
  "uniqueID"     INT8       DEFAULT unique_rowid(),
  PRIMARY KEY (timestamp, "uniqueID")
);`

	UITableSchema = `
CREATE TABLE system.ui (
	key           STRING PRIMARY KEY,
	value         BYTES,
	"lastUpdated" TIMESTAMP NOT NULL
);`

	JobsTableSchema = `
CREATE TABLE system.jobs (
	id                INT8      DEFAULT unique_rowid() PRIMARY KEY,
	status            STRING    NOT NULL,
	created           TIMESTAMP NOT NULL DEFAULT now(),
	payload           BYTES     NOT NULL,
	INDEX (status, created),
	FAMILY (id, status, created, payload)
);`

	// web_sessions are used to track authenticated user actions over stateless
	// connections, such as the cookie-based authentication used by the Admin
	// UI.
	// Design outlined in /docs/RFCS/web_session_login.rfc
	WebSessionsTableSchema = `
CREATE TABLE system.web_sessions (
	id             INT8       NOT NULL DEFAULT unique_rowid() PRIMARY KEY,
	"hashedSecret" BYTES      NOT NULL,
	username       STRING     NOT NULL,
	"createdAt"    TIMESTAMP  NOT NULL DEFAULT now(),
	"expiresAt"    TIMESTAMP  NOT NULL,
	"revokedAt"    TIMESTAMP,
	"lastUsedAt"   TIMESTAMP  NOT NULL DEFAULT now(),
	"auditInfo"    STRING,
	INDEX ("expiresAt"),
	INDEX ("createdAt"),
	FAMILY (id, "hashedSecret", username, "createdAt", "expiresAt", "revokedAt", "lastUsedAt", "auditInfo")
);`

	// table_statistics is used to track statistics collected about individual columns
	// or groups of columns from every table in the database. Each row contains the
	// number of distinct values of the column group and (optionally) a histogram if there
	// is only one column in columnIDs.
	//
	// Design outlined in /docs/RFCS/20170908_sql_optimizer_statistics.md
	TableStatisticsTableSchema = `
CREATE TABLE system.table_statistics (
	"tableID"       INT8       NOT NULL,
	"statisticID"   INT8       NOT NULL DEFAULT unique_rowid(),
	name            STRING,
	"columnIDs"     INT8[]     NOT NULL,
	"createdAt"     TIMESTAMP  NOT NULL DEFAULT now(),
	"rowCount"      INT8       NOT NULL,
	"distinctCount" INT8       NOT NULL,
	"nullCount"     INT8       NOT NULL,
	histogram       BYTES,
	PRIMARY KEY ("tableID", "statisticID"),
	FAMILY ("tableID", "statisticID", name, "columnIDs", "createdAt", "rowCount", "distinctCount", "nullCount", histogram)
);`

	// locations are used to map a locality specified by a node to geographic
	// latitude, longitude coordinates, specified as degrees.
	LocationsTableSchema = `
CREATE TABLE system.locations (
  "localityKey"   STRING,
  "localityValue" STRING,
  latitude        DECIMAL(18,15) NOT NULL,
  longitude       DECIMAL(18,15) NOT NULL,
  PRIMARY KEY ("localityKey", "localityValue"),
  FAMILY ("localityKey", "localityValue", latitude, longitude)
);`

	// role_members stores relationships between roles (role->role and role->user).
	RoleMembersTableSchema = `
CREATE TABLE system.role_members (
  "role"   STRING NOT NULL,
  "member" STRING NOT NULL,
  "isAdmin"  BOOL NOT NULL,
  PRIMARY KEY  ("role", "member"),
  INDEX ("role"),
  INDEX ("member")
);`

	// comments stores comments(database, table, column...).
	CommentsTableSchema = `
CREATE TABLE system.comments (
   type      INT NOT NULL,    -- type of object, to distinguish between db, table, column and others
   object_id INT NOT NULL,    -- object ID, this will be usually db/table desc ID
   sub_id    INT NOT NULL,    -- sub ID for columns inside table, 0 for pure table
   comment   STRING NOT NULL, -- the comment
   PRIMARY KEY (type, object_id, sub_id)
);`

	// statement_executions stores metrics about executed statements.
	StatementExecutionsTableSchema = `
CREATE TABLE system.statement_executions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id                 INT NOT NULL, -- the gateway node
    received_at             TIMESTAMP NOT NULL,
    statement               STRING NOT NULL,
    application_name        STRING NOT NULL,
    distributed             BOOL NOT NULL,
    optimized               BOOL NOT NULL,
    automatic_retry_count   INT NOT NULL,
    error                   STRING NULL, -- if execution failed, the error
    rows_affected           INT NOT NULL,
    parse_lat               INTERVAL NOT NULL,
    plan_lat                INTERVAL NOT NULL,
    run_lat                 INTERVAL NOT NULL,
    service_lat             INTERVAL NOT NULL
);`
)

func pk(name string) IndexDescriptor {
	return IndexDescriptor{
		Name:             "primary",
		ID:               1,
		Unique:           true,
		ColumnNames:      []string{name},
		ColumnDirections: singleASC,
		ColumnIDs:        singleID1,
	}
}

// SystemAllowedPrivileges describes the allowable privilege list for each
// system object. Super users (root and admin) must have exactly the specified privileges,
// other users must not exceed the specified privileges.
var SystemAllowedPrivileges = map[ID]privilege.List{
	keys.SystemDatabaseID:  privilege.ReadData,
	keys.NamespaceTableID:  privilege.ReadData,
	keys.DescriptorTableID: privilege.ReadData,
	keys.UsersTableID:      privilege.ReadWriteData,
	keys.ZonesTableID:      privilege.ReadWriteData,
	// We eventually want to migrate the table to appear read-only to force the
	// the use of a validating, logging accessor, so we'll go ahead and tolerate
	// read-only privs to make that migration possible later.
	keys.SettingsTableID:   privilege.ReadWriteData,
	keys.LeaseTableID:      privilege.ReadWriteData,
	keys.EventLogTableID:   privilege.ReadWriteData,
	keys.RangeEventTableID: privilege.ReadWriteData,
	keys.UITableID:         privilege.ReadWriteData,
	// IMPORTANT: CREATE|DROP|ALL privileges should always be denied or database
	// users will be able to modify system tables' schemas at will. CREATE and
	// DROP privileges are allowed on the above system tables for backwards
	// compatibility reasons only!
	keys.JobsTableID:            privilege.ReadWriteData,
	keys.WebSessionsTableID:     privilege.ReadWriteData,
	keys.TableStatisticsTableID: privilege.ReadWriteData,
	keys.LocationsTableID:       privilege.ReadWriteData,
	keys.RoleMembersTableID:     privilege.ReadWriteData,
	keys.CommentsTableID:        privilege.ReadWriteData,
	// TODO(couchand): This really should be read-only.
	keys.StatementExecutionsTableID: privilege.ReadWriteData,
}

// Helpers used to make some of the TableDescriptor literals below more concise.
var (
	colTypeBool      = ColumnType{SemanticType: ColumnType_BOOL}
	colTypeInt       = ColumnType{SemanticType: ColumnType_INT, VisibleType: ColumnType_BIGINT, Width: 64}
	colTypeInterval  = ColumnType{SemanticType: ColumnType_INTERVAL}
	colTypeString    = ColumnType{SemanticType: ColumnType_STRING}
	colTypeBytes     = ColumnType{SemanticType: ColumnType_BYTES}
	colTypeTimestamp = ColumnType{SemanticType: ColumnType_TIMESTAMP}
	colTypeUuid      = ColumnType{SemanticType: ColumnType_UUID}
	colTypeIntArray  = ColumnType{
		SemanticType:    ColumnType_ARRAY,
		ArrayContents:   &colTypeInt.SemanticType,
		VisibleType:     colTypeInt.VisibleType,
		Width:           colTypeInt.Width,
		ArrayDimensions: []int32{-1}}
	singleASC = []IndexDescriptor_Direction{IndexDescriptor_ASC}
	singleID1 = []ColumnID{1}
)

// These system config TableDescriptor literals should match the descriptor
// that would be produced by evaluating one of the above `CREATE TABLE`
// statements. See the `TestSystemTableLiterals` which checks that they do
// indeed match, and has suggestions on writing and maintaining them.
var (
	// SystemDB is the descriptor for the system database.
	SystemDB = DatabaseDescriptor{
		Name: "system",
		ID:   keys.SystemDatabaseID,
		// Assign max privileges to root user.
		Privileges: NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.SystemDatabaseID]),
	}

	// NamespaceTable is the descriptor for the namespace table.
	NamespaceTable = TableDescriptor{
		Name:     "namespace",
		ID:       keys.NamespaceTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "parentID", ID: 1, Type: colTypeInt},
			{Name: "name", ID: 2, Type: colTypeString},
			{Name: "id", ID: 3, Type: colTypeInt, Nullable: true},
		},
		NextColumnID: 4,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"parentID", "name"}, ColumnIDs: []ColumnID{1, 2}},
			{Name: "fam_3_id", ID: 3, ColumnNames: []string{"id"}, ColumnIDs: []ColumnID{3}, DefaultColumnID: 3},
		},
		NextFamilyID: 4,
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"parentID", "name"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 2},
		},
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.NamespaceTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	// DescriptorTable is the descriptor for the descriptor table.
	DescriptorTable = TableDescriptor{
		Name:       "descriptor",
		ID:         keys.DescriptorTableID,
		Privileges: NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.DescriptorTableID]),
		ParentID:   keys.SystemDatabaseID,
		Version:    1,
		Columns: []ColumnDescriptor{
			{Name: "id", ID: 1, Type: colTypeInt},
			{Name: "descriptor", ID: 2, Type: colTypeBytes, Nullable: true},
		},
		NextColumnID: 3,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"id"}, ColumnIDs: singleID1},
			{Name: "fam_2_descriptor", ID: 2, ColumnNames: []string{"descriptor"}, ColumnIDs: []ColumnID{2}, DefaultColumnID: 2},
		},
		PrimaryIndex:   pk("id"),
		NextFamilyID:   3,
		NextIndexID:    2,
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	falseBoolString = "false"

	// UsersTable is the descriptor for the users table.
	UsersTable = TableDescriptor{
		Name:     "users",
		ID:       keys.UsersTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "username", ID: 1, Type: colTypeString},
			{Name: "hashedPassword", ID: 2, Type: colTypeBytes, Nullable: true},
			{Name: "isRole", ID: 3, Type: colTypeBool, DefaultExpr: &falseBoolString},
		},
		NextColumnID: 4,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"username"}, ColumnIDs: singleID1},
			{Name: "fam_2_hashedPassword", ID: 2, ColumnNames: []string{"hashedPassword"}, ColumnIDs: []ColumnID{2}, DefaultColumnID: 2},
			{Name: "fam_3_isRole", ID: 3, ColumnNames: []string{"isRole"}, ColumnIDs: []ColumnID{3}, DefaultColumnID: 3},
		},
		PrimaryIndex:   pk("username"),
		NextFamilyID:   4,
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.UsersTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	// ZonesTable is the descriptor for the zones table.
	ZonesTable = TableDescriptor{
		Name:     "zones",
		ID:       keys.ZonesTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "id", ID: 1, Type: colTypeInt},
			{Name: "config", ID: keys.ZonesTableConfigColumnID, Type: colTypeBytes, Nullable: true},
		},
		NextColumnID: 3,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"id"}, ColumnIDs: singleID1},
			{Name: "fam_2_config", ID: 2, ColumnNames: []string{"config"}, ColumnIDs: []ColumnID{2}, DefaultColumnID: 2},
		},
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               keys.ZonesTablePrimaryIndexID,
			Unique:           true,
			ColumnNames:      []string{"id"},
			ColumnDirections: singleASC,
			ColumnIDs:        []ColumnID{keys.ZonesTablePrimaryIndexID},
		},
		NextFamilyID:   3,
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.ZonesTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}
	// SettingsTable is the descriptor for the jobs table.
	SettingsTable = TableDescriptor{
		Name:     "settings",
		ID:       keys.SettingsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "name", ID: 1, Type: colTypeString},
			{Name: "value", ID: 2, Type: colTypeString},
			{Name: "lastUpdated", ID: 3, Type: colTypeTimestamp, DefaultExpr: &nowString},
			{Name: "valueType", ID: 4, Type: colTypeString, Nullable: true},
		},
		NextColumnID: 5,
		Families: []ColumnFamilyDescriptor{
			{
				Name:        "fam_0_name_value_lastUpdated_valueType",
				ID:          0,
				ColumnNames: []string{"name", "value", "lastUpdated", "valueType"},
				ColumnIDs:   []ColumnID{1, 2, 3, 4},
			},
		},
		NextFamilyID:   1,
		PrimaryIndex:   pk("name"),
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.SettingsTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}
)

// These system TableDescriptor literals should match the descriptor that
// would be produced by evaluating one of the above `CREATE TABLE` statements
// for system tables that are not system config tables. See the
// `TestSystemTableLiterals` which checks that they do indeed match, and has
// suggestions on writing and maintaining them.
var (
	// LeaseTable is the descriptor for the leases table.
	LeaseTable = TableDescriptor{
		Name:     "lease",
		ID:       keys.LeaseTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "descID", ID: 1, Type: colTypeInt},
			{Name: "version", ID: 2, Type: colTypeInt},
			{Name: "nodeID", ID: 3, Type: colTypeInt},
			{Name: "expiration", ID: 4, Type: colTypeTimestamp},
		},
		NextColumnID: 5,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"descID", "version", "nodeID", "expiration"}, ColumnIDs: []ColumnID{1, 2, 3, 4}},
		},
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"descID", "version", "expiration", "nodeID"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC, IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 2, 4, 3},
		},
		NextFamilyID:   1,
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.LeaseTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	uuidV4String = "uuid_v4()"

	// EventLogTable is the descriptor for the event log table.
	EventLogTable = TableDescriptor{
		Name:     "eventlog",
		ID:       keys.EventLogTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "timestamp", ID: 1, Type: colTypeTimestamp},
			{Name: "eventType", ID: 2, Type: colTypeString},
			{Name: "targetID", ID: 3, Type: colTypeInt},
			{Name: "reportingID", ID: 4, Type: colTypeInt},
			{Name: "info", ID: 5, Type: colTypeString, Nullable: true},
			{Name: "uniqueID", ID: 6, Type: colTypeBytes, DefaultExpr: &uuidV4String},
		},
		NextColumnID: 7,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"timestamp", "uniqueID"}, ColumnIDs: []ColumnID{1, 6}},
			{Name: "fam_2_eventType", ID: 2, ColumnNames: []string{"eventType"}, ColumnIDs: []ColumnID{2}, DefaultColumnID: 2},
			{Name: "fam_3_targetID", ID: 3, ColumnNames: []string{"targetID"}, ColumnIDs: []ColumnID{3}, DefaultColumnID: 3},
			{Name: "fam_4_reportingID", ID: 4, ColumnNames: []string{"reportingID"}, ColumnIDs: []ColumnID{4}, DefaultColumnID: 4},
			{Name: "fam_5_info", ID: 5, ColumnNames: []string{"info"}, ColumnIDs: []ColumnID{5}, DefaultColumnID: 5},
		},
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"timestamp", "uniqueID"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 6},
		},
		NextFamilyID:   6,
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.EventLogTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	uniqueRowIDString = "unique_rowid()"

	// RangeEventTable is the descriptor for the range log table.
	RangeEventTable = TableDescriptor{
		Name:     "rangelog",
		ID:       keys.RangeEventTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "timestamp", ID: 1, Type: colTypeTimestamp},
			{Name: "rangeID", ID: 2, Type: colTypeInt},
			{Name: "storeID", ID: 3, Type: colTypeInt},
			{Name: "eventType", ID: 4, Type: colTypeString},
			{Name: "otherRangeID", ID: 5, Type: colTypeInt, Nullable: true},
			{Name: "info", ID: 6, Type: colTypeString, Nullable: true},
			{Name: "uniqueID", ID: 7, Type: colTypeInt, DefaultExpr: &uniqueRowIDString},
		},
		NextColumnID: 8,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"timestamp", "uniqueID"}, ColumnIDs: []ColumnID{1, 7}},
			{Name: "fam_2_rangeID", ID: 2, ColumnNames: []string{"rangeID"}, ColumnIDs: []ColumnID{2}, DefaultColumnID: 2},
			{Name: "fam_3_storeID", ID: 3, ColumnNames: []string{"storeID"}, ColumnIDs: []ColumnID{3}, DefaultColumnID: 3},
			{Name: "fam_4_eventType", ID: 4, ColumnNames: []string{"eventType"}, ColumnIDs: []ColumnID{4}, DefaultColumnID: 4},
			{Name: "fam_5_otherRangeID", ID: 5, ColumnNames: []string{"otherRangeID"}, ColumnIDs: []ColumnID{5}, DefaultColumnID: 5},
			{Name: "fam_6_info", ID: 6, ColumnNames: []string{"info"}, ColumnIDs: []ColumnID{6}, DefaultColumnID: 6},
		},
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"timestamp", "uniqueID"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 7},
		},
		NextFamilyID:   7,
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.RangeEventTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	// UITable is the descriptor for the ui table.
	UITable = TableDescriptor{
		Name:     "ui",
		ID:       keys.UITableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "key", ID: 1, Type: colTypeString},
			{Name: "value", ID: 2, Type: colTypeBytes, Nullable: true},
			{Name: "lastUpdated", ID: 3, Type: ColumnType{SemanticType: ColumnType_TIMESTAMP}},
		},
		NextColumnID: 4,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"key"}, ColumnIDs: singleID1},
			{Name: "fam_2_value", ID: 2, ColumnNames: []string{"value"}, ColumnIDs: []ColumnID{2}, DefaultColumnID: 2},
			{Name: "fam_3_lastUpdated", ID: 3, ColumnNames: []string{"lastUpdated"}, ColumnIDs: []ColumnID{3}, DefaultColumnID: 3},
		},
		NextFamilyID:   4,
		PrimaryIndex:   pk("key"),
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.UITableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	nowString = "now():::TIMESTAMP"

	// JobsTable is the descriptor for the jobs table.
	JobsTable = TableDescriptor{
		Name:     "jobs",
		ID:       keys.JobsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "id", ID: 1, Type: colTypeInt, DefaultExpr: &uniqueRowIDString},
			{Name: "status", ID: 2, Type: colTypeString},
			{Name: "created", ID: 3, Type: colTypeTimestamp, DefaultExpr: &nowString},
			{Name: "payload", ID: 4, Type: colTypeBytes},
		},
		NextColumnID: 5,
		Families: []ColumnFamilyDescriptor{
			{
				Name:        "fam_0_id_status_created_payload",
				ID:          0,
				ColumnNames: []string{"id", "status", "created", "payload"},
				ColumnIDs:   []ColumnID{1, 2, 3, 4},
			},
		},
		NextFamilyID: 1,
		PrimaryIndex: pk("id"),
		Indexes: []IndexDescriptor{
			{
				Name:             "jobs_status_created_idx",
				ID:               2,
				Unique:           false,
				ColumnNames:      []string{"status", "created"},
				ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
				ColumnIDs:        []ColumnID{2, 3},
				ExtraColumnIDs:   []ColumnID{1},
			},
		},
		NextIndexID:    3,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.JobsTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	// WebSessions table to authenticate sessions over stateless connections.
	WebSessionsTable = TableDescriptor{
		Name:     "web_sessions",
		ID:       keys.WebSessionsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "id", ID: 1, Type: colTypeInt, DefaultExpr: &uniqueRowIDString},
			{Name: "hashedSecret", ID: 2, Type: colTypeBytes},
			{Name: "username", ID: 3, Type: colTypeString},
			{Name: "createdAt", ID: 4, Type: colTypeTimestamp, DefaultExpr: &nowString},
			{Name: "expiresAt", ID: 5, Type: colTypeTimestamp},
			{Name: "revokedAt", ID: 6, Type: colTypeTimestamp, Nullable: true},
			{Name: "lastUsedAt", ID: 7, Type: colTypeTimestamp, DefaultExpr: &nowString},
			{Name: "auditInfo", ID: 8, Type: colTypeString, Nullable: true},
		},
		NextColumnID: 9,
		Families: []ColumnFamilyDescriptor{
			{
				Name: "fam_0_id_hashedSecret_username_createdAt_expiresAt_revokedAt_lastUsedAt_auditInfo",
				ID:   0,
				ColumnNames: []string{
					"id",
					"hashedSecret",
					"username",
					"createdAt",
					"expiresAt",
					"revokedAt",
					"lastUsedAt",
					"auditInfo",
				},
				ColumnIDs: []ColumnID{1, 2, 3, 4, 5, 6, 7, 8},
			},
		},
		NextFamilyID: 1,
		PrimaryIndex: pk("id"),
		Indexes: []IndexDescriptor{
			{
				Name:             "web_sessions_expiresAt_idx",
				ID:               2,
				Unique:           false,
				ColumnNames:      []string{"expiresAt"},
				ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC},
				ColumnIDs:        []ColumnID{5},
				ExtraColumnIDs:   []ColumnID{1},
			},
			{
				Name:             "web_sessions_createdAt_idx",
				ID:               3,
				Unique:           false,
				ColumnNames:      []string{"createdAt"},
				ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC},
				ColumnIDs:        []ColumnID{4},
				ExtraColumnIDs:   []ColumnID{1},
			},
		},
		NextIndexID:    4,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.WebSessionsTableID]),
		NextMutationID: 1,
		FormatVersion:  3,
	}

	// TableStatistics table to hold statistics about columns and column groups.
	TableStatisticsTable = TableDescriptor{
		Name:     "table_statistics",
		ID:       keys.TableStatisticsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "tableID", ID: 1, Type: colTypeInt},
			{Name: "statisticID", ID: 2, Type: colTypeInt, DefaultExpr: &uniqueRowIDString},
			{Name: "name", ID: 3, Type: colTypeString, Nullable: true},
			{Name: "columnIDs", ID: 4, Type: colTypeIntArray},
			{Name: "createdAt", ID: 5, Type: colTypeTimestamp, DefaultExpr: &nowString},
			{Name: "rowCount", ID: 6, Type: colTypeInt},
			{Name: "distinctCount", ID: 7, Type: colTypeInt},
			{Name: "nullCount", ID: 8, Type: colTypeInt},
			{Name: "histogram", ID: 9, Type: colTypeBytes, Nullable: true},
		},
		NextColumnID: 10,
		Families: []ColumnFamilyDescriptor{
			{
				Name: "fam_0_tableID_statisticID_name_columnIDs_createdAt_rowCount_distinctCount_nullCount_histogram",
				ID:   0,
				ColumnNames: []string{
					"tableID",
					"statisticID",
					"name",
					"columnIDs",
					"createdAt",
					"rowCount",
					"distinctCount",
					"nullCount",
					"histogram",
				},
				ColumnIDs: []ColumnID{1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
		NextFamilyID: 1,
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"tableID", "statisticID"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 2},
		},
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.TableStatisticsTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	latLonDecimal = ColumnType{
		SemanticType: ColumnType_DECIMAL,
		Precision:    18,
		Width:        15,
	}

	// LocationsTable is the descriptor for the locations table.
	LocationsTable = TableDescriptor{
		Name:     "locations",
		ID:       keys.LocationsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "localityKey", ID: 1, Type: colTypeString},
			{Name: "localityValue", ID: 2, Type: colTypeString},
			{Name: "latitude", ID: 3, Type: latLonDecimal},
			{Name: "longitude", ID: 4, Type: latLonDecimal},
		},
		NextColumnID: 5,
		Families: []ColumnFamilyDescriptor{
			{
				Name:        "fam_0_localityKey_localityValue_latitude_longitude",
				ID:          0,
				ColumnNames: []string{"localityKey", "localityValue", "latitude", "longitude"},
				ColumnIDs:   []ColumnID{1, 2, 3, 4},
			},
		},
		NextFamilyID: 1,
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"localityKey", "localityValue"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 2},
		},
		NextIndexID:    2,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.LocationsTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	// RoleMembersTable is the descriptor for the role_members table.
	RoleMembersTable = TableDescriptor{
		Name:     "role_members",
		ID:       keys.RoleMembersTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "role", ID: 1, Type: colTypeString},
			{Name: "member", ID: 2, Type: colTypeString},
			{Name: "isAdmin", ID: 3, Type: colTypeBool},
		},
		NextColumnID: 4,
		Families: []ColumnFamilyDescriptor{
			{
				Name:        "primary",
				ID:          0,
				ColumnNames: []string{"role", "member"},
				ColumnIDs:   []ColumnID{1, 2},
			},
			{
				Name:            "fam_3_isAdmin",
				ID:              3,
				ColumnNames:     []string{"isAdmin"},
				ColumnIDs:       []ColumnID{3},
				DefaultColumnID: 3,
			},
		},
		NextFamilyID: 4,
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"role", "member"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 2},
		},
		Indexes: []IndexDescriptor{
			{
				Name:             "role_members_role_idx",
				ID:               2,
				Unique:           false,
				ColumnNames:      []string{"role"},
				ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC},
				ColumnIDs:        []ColumnID{1},

				ExtraColumnIDs: []ColumnID{2},
			},
			{
				Name:             "role_members_member_idx",
				ID:               3,
				Unique:           false,
				ColumnNames:      []string{"member"},
				ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC},
				ColumnIDs:        []ColumnID{2},
				ExtraColumnIDs:   []ColumnID{1},
			},
		},
		NextIndexID:    4,
		Privileges:     NewCustomSuperuserPrivilegeDescriptor(SystemAllowedPrivileges[keys.RoleMembersTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	// CommentsTable is the descriptor for the comments table.
	CommentsTable = TableDescriptor{
		Name:     "comments",
		ID:       keys.CommentsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "type", ID: 1, Type: colTypeInt},
			{Name: "object_id", ID: 2, Type: colTypeInt},
			{Name: "sub_id", ID: 3, Type: colTypeInt},
			{Name: "comment", ID: 4, Type: colTypeString},
		},
		NextColumnID: 5,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{"type", "object_id", "sub_id"}, ColumnIDs: []ColumnID{1, 2, 3}},
			{Name: "fam_4_comment", ID: 4, ColumnNames: []string{"comment"}, ColumnIDs: []ColumnID{4}, DefaultColumnID: 4},
		},
		NextFamilyID: 5,
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"type", "object_id", "sub_id"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC, IndexDescriptor_ASC, IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1, 2, 3},
		},
		NextIndexID:    2,
		Privileges:     newCommentPrivilegeDescriptor(SystemAllowedPrivileges[keys.CommentsTableID]),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}

	genRandomUUIDString = "gen_random_uuid()"

	// StatementExecutionsTable is the descriptor for the statement_executions table.
	StatementExecutionsTable = TableDescriptor{
		Name:     "statement_executions",
		ID:       keys.StatementExecutionsTableID,
		ParentID: keys.SystemDatabaseID,
		Version:  1,
		Columns: []ColumnDescriptor{
			{Name: "id", ID: 1, Type: colTypeUuid, DefaultExpr: &genRandomUUIDString},
			{Name: "node_id", ID: 2, Type: colTypeInt},
			{Name: "received_at", ID: 3, Type: colTypeTimestamp},
			{Name: "statement", ID: 4, Type: colTypeString},
			{Name: "application_name", ID: 5, Type: colTypeString},
			{Name: "distributed", ID: 6, Type: colTypeBool},
			{Name: "optimized", ID: 7, Type: colTypeBool},
			{Name: "automatic_retry_count", ID: 8, Type: colTypeInt},
			{Name: "error", ID: 9, Type: colTypeString, Nullable: true},
			{Name: "rows_affected", ID: 10, Type: colTypeInt},
			{Name: "parse_lat", ID: 11, Type: colTypeInterval},
			{Name: "plan_lat", ID: 12, Type: colTypeInterval},
			{Name: "run_lat", ID: 13, Type: colTypeInterval},
			{Name: "service_lat", ID: 14, Type: colTypeInterval},
		},
		NextColumnID: 15,
		Families: []ColumnFamilyDescriptor{
			{Name: "primary", ID: 0, ColumnNames: []string{
				"id", "node_id", "received_at", "statement",
				"application_name", "distributed", "optimized",
				"automatic_retry_count", "error", "rows_affected",
				"parse_lat", "plan_lat", "run_lat", "service_lat"},
				ColumnIDs: []ColumnID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}},
		},
		NextFamilyID: 1,
		PrimaryIndex: IndexDescriptor{
			Name:             "primary",
			ID:               1,
			Unique:           true,
			ColumnNames:      []string{"id"},
			ColumnDirections: []IndexDescriptor_Direction{IndexDescriptor_ASC},
			ColumnIDs:        []ColumnID{1},
		},
		NextIndexID: 2,
		Privileges: NewCustomSuperuserPrivilegeDescriptor(
			SystemAllowedPrivileges[keys.StatementExecutionsTableID],
		),
		FormatVersion:  InterleavedFormatVersion,
		NextMutationID: 1,
	}
)

// Create a kv pair for the zone config for the given key and config value.
func createZoneConfigKV(keyID int, zoneConfig *config.ZoneConfig) roachpb.KeyValue {
	value := roachpb.Value{}
	if err := value.SetProto(zoneConfig); err != nil {
		panic(fmt.Sprintf("could not marshal ZoneConfig for ID: %d: %s", keyID, err))
	}
	return roachpb.KeyValue{
		Key:   config.MakeZoneKey(uint32(keyID)),
		Value: value,
	}
}

// addSystemDatabaseToSchema populates the supplied MetadataSchema with the
// System database and its tables. The descriptors for these objects exist
// statically in this file, but a MetadataSchema can be used to persist these
// descriptors to the cockroach store.
func addSystemDatabaseToSchema(target *MetadataSchema) {
	// Add system database.
	target.AddDescriptor(keys.RootNamespaceID, &SystemDB)

	// Add system config tables.
	target.AddDescriptor(keys.SystemDatabaseID, &NamespaceTable)
	target.AddDescriptor(keys.SystemDatabaseID, &DescriptorTable)
	target.AddDescriptor(keys.SystemDatabaseID, &UsersTable)
	target.AddDescriptor(keys.SystemDatabaseID, &ZonesTable)

	// Add all the other system tables.
	target.AddDescriptor(keys.SystemDatabaseID, &LeaseTable)
	target.AddDescriptor(keys.SystemDatabaseID, &EventLogTable)
	target.AddDescriptor(keys.SystemDatabaseID, &RangeEventTable)
	target.AddDescriptor(keys.SystemDatabaseID, &UITable)
	target.AddDescriptor(keys.SystemDatabaseID, &JobsTable)
	target.AddDescriptor(keys.SystemDatabaseID, &SettingsTable)
	target.AddDescriptor(keys.SystemDatabaseID, &WebSessionsTable)

	// Tables introduced in 2.0, added here for 2.1.
	target.AddDescriptor(keys.SystemDatabaseID, &TableStatisticsTable)
	target.AddDescriptor(keys.SystemDatabaseID, &LocationsTable)
	target.AddDescriptor(keys.SystemDatabaseID, &RoleMembersTable)

	// The CommentsTable has been introduced in 2.2. It was added here since it
	// was introduced, but it's also created as a migration for older clusters.
	target.AddDescriptor(keys.SystemDatabaseID, &CommentsTable)

	target.AddSplitIDs(keys.PseudoTableIDs...)

	// Adding a new system table? It should be added here to the metadata schema,
	// and also created as a migration for older cluster. The includedInBootstrap
	// field should be set on the migration.

	// Default zone config entry.
	zoneConf := config.DefaultZoneConfig()
	target.otherKV = append(target.otherKV, createZoneConfigKV(keys.RootNamespaceID, &zoneConf))

	systemZoneConf := config.DefaultSystemZoneConfig()
	metaRangeZoneConf := config.DefaultSystemZoneConfig()
	jobsZoneConf := config.DefaultSystemZoneConfig()
	livenessZoneConf := config.DefaultSystemZoneConfig()

	// .meta zone config entry with a shorter GC time.
	metaRangeZoneConf.GC.TTLSeconds = 60 * 60 // 1h
	target.otherKV = append(target.otherKV, createZoneConfigKV(keys.MetaRangesID, &metaRangeZoneConf))

	// Jobs zone config entry with a shorter GC time.
	jobsZoneConf.GC.TTLSeconds = 10 * 60 // 10m
	target.otherKV = append(target.otherKV, createZoneConfigKV(keys.JobsTableID, &jobsZoneConf))

	// Liveness zone config entry with a shorter GC time.
	livenessZoneConf.GC.TTLSeconds = 10 * 60 // 10m
	target.otherKV = append(target.otherKV, createZoneConfigKV(keys.LivenessRangesID, &livenessZoneConf))
	target.otherKV = append(target.otherKV, createZoneConfigKV(keys.SystemRangesID, &systemZoneConf))
	target.otherKV = append(target.otherKV, createZoneConfigKV(keys.SystemDatabaseID, &systemZoneConf))
}

// IsSystemConfigID returns whether this ID is for a system config object.
func IsSystemConfigID(id ID) bool {
	return id > 0 && id <= keys.MaxSystemConfigDescID
}

// IsReservedID returns whether this ID is for any system object.
func IsReservedID(id ID) bool {
	return id > 0 && id <= keys.MaxReservedDescID
}

// newCommentPrivilegeDescriptor returns a privilege descriptor for comment table
func newCommentPrivilegeDescriptor(priv privilege.List) *PrivilegeDescriptor {
	return &PrivilegeDescriptor{
		Users: []UserPrivileges{
			{
				User:       AdminRole,
				Privileges: priv.ToBitField(),
			},
			{
				User:       PublicRole,
				Privileges: priv.ToBitField(),
			},
			{
				User:       security.RootUser,
				Privileges: priv.ToBitField(),
			},
		},
	}
}
