import React from "react";

import { LineGraph } from "src/views/cluster/components/linegraph";
import { Metric, Axis } from "src/views/shared/components/metricQuery";
import { DashboardConfig } from "src/util/charts";

import { GraphDashboardProps } from "./dashboardUtils";

export const charts: DashboardConfig = {
  id: "nodes.requests",
  charts: [
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.node.requests.slow.distsender", title: "Slow Distsender Requests" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.requests.slow.raft", title: "Slow Raft Proposals" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.requests.slow.lease", title: "Slow Lease Acquisitions" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.requests.slow.commandqueue", title: "Slow Command Queue Entries" },
      ],
    },
  ],
};

export default function (props: GraphDashboardProps) {
  const { storeSources, nodeSources } = props;

  return [
    <LineGraph title="Slow Distsender Requests" sources={nodeSources}>
      <Axis>
        <Metric name="cr.node.requests.slow.distsender" title="Slow Distsender Requests" nonNegativeRate />
      </Axis>
    </LineGraph>,

    <LineGraph title="Slow Raft Proposals" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.requests.slow.raft" title="Slow Raft Proposals" nonNegativeRate />
      </Axis>
    </LineGraph>,

    <LineGraph title="Slow Lease Acquisitions" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.requests.slow.lease" title="Slow Lease Acquisitions" nonNegativeRate />
      </Axis>
    </LineGraph>,

    <LineGraph title="Slow Command Queue Entries" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.requests.slow.commandqueue" title="Slow Command Queue Entries" nonNegativeRate />
      </Axis>
    </LineGraph>,
  ];
}
