import moment from "moment";
import React from "react";
import { connect } from "react-redux";

import { hoverStateSelector, HoverState } from "src/redux/hover";
import { AdminUIState } from "src/redux/state";

interface HoverThingProps {
  hoverState: HoverState;
}

class HoverThing extends React.Component<HoverThingProps, {}> {
  render() {
    const { hoverChart, hoverTime } = this.props.hoverState;

    if (!hoverChart) {
      return null;
    }

    const metrics = [
      { title: "Foo", color: "steelblue", value: "0.000" },
      { title: "Bar", color: "red", value: "1.000" },
    ];

    const time = moment(hoverTime).utc();
    return (
      <section className="hover-thing">
        <div className="hover-thing__time">
          { time.format("HH:mm:ss") }
          <span className="legend-subtext"> on </span>
          { time.format("MMM Do, YYYY") }
        </div>
        {
          metrics.map((metric) => (
            <div className="hover-thing__metric">
              <svg width={9} height={9}><circle cx={4.5} cy={4.5} r={3.5} fill={metric.color} /></svg>
              <span className="label">{ metric.title }</span>
              <span className="value">{ metric.value }</span>
            </div>
          ))
        }
      </section>
    );
  }
}

function mapStateToProps(state: AdminUIState) {
  return {
    hoverState: hoverStateSelector(state),
  };
}

export default connect(mapStateToProps)(HoverThing);
