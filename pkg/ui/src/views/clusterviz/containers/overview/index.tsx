import React from "react";
import { connect } from "react-redux";

import { refreshNodes, refreshLiveness } from "src/redux/apiReducers";
import { nodesSummarySelector, NodesSummary } from "src/redux/nodes";
import { AdminUIState } from "src/redux/state";
import { initNodeCanvas, updateNodeCanvas } from "./sim";
import "./sim.css";

interface ClusterVizProps {
    nodesSummary: NodesSummary;
    statusesValid: boolean;
    refreshNodes: typeof refreshNodes;
    refreshLiveness: typeof refreshLiveness;
}

export class ClusterVizMain extends React.Component<ClusterVizProps, {}> {
    svgEl: SVGElement;
    model: any;

    componentWillMount() {
        // Refresh nodes status query when mounting.
        this.props.refreshNodes();
        this.props.refreshLiveness();
    }

    componentWillReceiveProps(props: any) {
        // Refresh nodes status query when props are received; this will immediately
        // trigger a new request if previous results are invalidated.
        props.refreshNodes();
        props.refreshLiveness();
    }

    componentDidMount() {
        this.model = initNodeCanvas(this.svgEl);
        this.drawNodeCanvas();
    }

    componentDidUpdate() {
        this.drawNodeCanvas();
    }

    drawNodeCanvas = () => {
        // If the document is not visible (e.g. if the window is minimized) we don't
        // attempt to redraw the chart. Redrawing the chart uses
        // requestAnimationFrame, which isn't called when the tab is in the
        // background, and is then apparently queued up and called en masse when the
        // tab re-enters the foreground. This check prevents the issue in #8896
        // where switching to a tab with the graphs page open that had been in the
        // background caused the UI to run out of memory and either lag or crash.
        // NOTE: This might not work on Android:
        // http://caniuse.com/#feat=pagevisibility
        if (!document.hidden && this.props.nodesSummary.nodeStatuses != null) {
            updateNodeCanvas(this.model, this.props.nodesSummary);
        }
    }

    render() {
        return (
            <svg className="cluster-viz" ref={svg => this.svgEl = svg}/>
        );
    }
}

export default connect(
    (state: AdminUIState) => ({
        nodesSummary: nodesSummarySelector(state),
        statusesValid: state.cachedData.nodes.valid && state.cachedData.liveness.valid,
    }),
    {
        refreshNodes,
        refreshLiveness,
    },
)(ClusterVizMain);
