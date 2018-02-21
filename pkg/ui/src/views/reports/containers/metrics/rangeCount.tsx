import React from "react";
import { connect } from "react-redux";

import spinner from "assets/spinner.gif";
import { refreshNodes } from "src/redux/apiReducers";
import { CachedDataReducerState } from "src/redux/cachedDataReducer";
import { selectNodeRequestStatus } from "src/redux/nodes";
import { AdminUIState } from "src/redux/state";
import Loading from "src/views/shared/components/loading";

interface RangeCountProps {
  nodeStatus: CachedDataReducerState<any>;
  refreshNodes: typeof refreshNodes;
}

class RangeCount extends React.Component<RangeCountProps, {}> {
  static title() {
    return <h1>Range Count</h1>;
  }

  componentWillMount() {
    this.props.refreshNodes();
  }

  componentWillReceiveProps(props: RangeCountProps) {
    props.refreshNodes();
  }

  render() {
    return (
      <Loading
        loading={ !this.props.nodeStatus.valid }
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
                <td>Nodes</td>
                <td>node_id</td>
                <td>####</td>
              </tr>
              <tr>
                <td>TableStats</td>
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

function mapStateToProps(state: AdminUIState) {
  return {
    nodeStatus: selectNodeRequestStatus(state),
  };
}

const actions = {
  refreshNodes,
};

export default connect(mapStateToProps, actions)(RangeCount);
