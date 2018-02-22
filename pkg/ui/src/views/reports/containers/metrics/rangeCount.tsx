import _ from "lodash";
import React from "react";
import { connect } from "react-redux";

import spinner from "assets/spinner.gif";
import {
  refreshNodes,
  refreshLiveness,
  refreshDatabases,
  refreshDatabaseDetails,
  refreshTableDetails,
  refreshTableStats,
} from "src/redux/apiReducers";
import { CachedDataReducerState } from "src/redux/cachedDataReducer";
import {
  selectNodeRequestStatus,
  selectLivenessRequestStatus,
  nodesSummarySelector,
  NodesSummary
} from "src/redux/nodes";
import { AdminUIState } from "src/redux/state";
import { NodeStatus$Properties, MetricConstants } from "src/util/proto";
import { databaseDetails, tableInfos } from "src/views/databases/containers/databaseSummary";
import Loading from "src/views/shared/components/loading";

interface RangeCountProps {
  nodeStatus: CachedDataReducerState<any>;
  livenessStatus: CachedDataReducerState<any>;
  nodesSummary: NodesSummary,
  refreshNodes: typeof refreshNodes;
  refreshLiveness: typeof refreshLiveness;
}

class RangeCount extends React.Component<RangeCountProps, {}> {
  static title() {
    return <h1>Range Count</h1>;
  }

  componentWillMount() {
    this.props.refreshNodes();
    this.props.refreshLiveness();
  }

  componentWillReceiveProps(props: RangeCountProps) {
    props.refreshNodes();
    props.refreshLiveness();
  }

  render() {
    const { nodeSums, nodeStatuses } = this.props.nodesSummary;
    const loading = !this.props.nodeStatus.valid || !this.props.livenessStatus.valid;

    return (
      <Loading
        loading={ loading }
        className="loading-image loading-image__spinner-left"
        image={ spinner }
      >
        <section className="section">
          <table>
            <thead>
              <tr>
                <th>Metric</th>
                <th>Source</th>
                <th>Count</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <th>Nodes</th>
                <td>total</td>
                <td>{ nodeSums.totalRanges }</td>
              </tr>
              {
                _.map(nodeStatuses, (node) => (
                  <tr>
                    <td></td>
                    <td>{ node.desc.node_id }</td>
                    <td>{ node.metrics[MetricConstants.ranges] }</td>
                  </tr>
                ))
              }
              <tr>
                <th>TableStats</th>
                <td>tableName</td>
                <td>####</td>
              </tr>
            </tbody>
          </table>
        </section>
      </Loading>
    );
  }
}

/*
refreshDatabaseInfo() {
  refreshDatabases()
    .then()

  loadTableDetails(props = this.props) {
    if (props.tableInfos && props.tableInfos.length > 0) {
      _.each(props.tableInfos, (tblInfo) => {
        if (_.isUndefined(tblInfo.numColumns)) {
          props.refreshTableDetails(new protos.cockroach.server.serverpb.TableDetailsRequest({
            database: props.name,
            table: tblInfo.name,
          }));
        }
        if (_.isUndefined(tblInfo.physicalSize)) {
          props.refreshTableStats(new protos.cockroach.server.serverpb.TableStatsRequest({
            database: props.name,
            table: tblInfo.name,
          }));
        }
      });
    }
  }

  // Refresh when the component is mounted.
  componentWillMount() {
    this.props.refreshDatabaseDetails(new protos.cockroach.server.serverpb.DatabaseDetailsRequest({ database: this.props.name }));
    this.loadTableDetails();
  }
}

function selectDatabaseInfo(state: AdminUIState) {
  const databases = databaseDetails(state);
  const result = {};

  const tableNames = dbDetails[dbName] && dbDetails[dbName].data && dbDetails[dbName].data.table_names;
  _.forEach(databases, (database, dbName) => {
    result[dbName]
  });

  return result;
}
*/

function mapStateToProps(state: AdminUIState) {
  return {
    nodeStatus: selectNodeRequestStatus(state),
    livenessStatus: selectLivenessRequestStatus(state),
    nodesSummary: nodesSummarySelector(state),
//  databases: selectDatabaseInfo(state),
  };
}

const actions = {
  refreshNodes,
  refreshLiveness,
};

export default connect(mapStateToProps, actions)(RangeCount);
