import React from "react";

import { LineGraph } from "src/views/cluster/components/linegraph";
import { Metric, Axis, AxisUnits } from "src/views/shared/components/metricQuery";
import { DashboardConfig } from "src/util/charts";

import { GraphDashboardProps } from "./dashboardUtils";

export const charts: DashboardConfig = {
  id: "nodes.queues",
  charts: [
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.gc.process.failure", title: "GC" },
        { name: "cr.store.queue.replicagc.process.failure", title: "Replica GC" },
        { name: "cr.store.queue.replicate.process.failure", title: "Replication" },
        { name: "cr.store.queue.split.process.failure", title: "Split" },
        { name: "cr.store.queue.consistency.process.failure", title: "Consistency" },
        { name: "cr.store.queue.raftlog.process.failure", title: "Raft Log" },
        { name: "cr.store.queue.tsmaintenance.process.failure", title: "Time Series Maintenance" },
      ],
    },
    {
      type: "metrics",
      measure: "duration",
      metrics: [
        { name: "cr.store.queue.gc.processingnanos", title: "GC" },
        { name: "cr.store.queue.replicagc.processingnanos", title: "Replica GC" },
        { name: "cr.store.queue.replicate.processingnanos", title: "Replication" },
        { name: "cr.store.queue.split.processingnanos", title: "Split" },
        { name: "cr.store.queue.consistency.processingnanos", title: "Consistency" },
        { name: "cr.store.queue.raftlog.processingnanos", title: "Raft Log" },
        { name: "cr.store.queue.tsmaintenance.processingnanos", title: "Time Series Maintenance" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.replicagc.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.replicagc.pending", title: "Pending Actions" },
        { name: "cr.store.queue.replicagc.removereplica", title: "Replicas Removed / sec" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.replicate.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.replicate.pending", title: "Pending Actions" },
        { name: "cr.store.queue.replicate.addreplica", title: "Replicas Added / sec" },
        { name: "cr.store.queue.replicate.removereplica", title: "Replicas Removed / sec" },
        { name: "cr.store.queue.replicate.removedeadreplica", title: "Dead Replicas Removed / sec" },
        { name: "cr.store.queue.replicate.rebalancereplica", title: "Replicas Rebalanced / sec" },
        { name: "cr.store.queue.replicate.transferlease", title: "Leases Transferred / sec" },
        { name: "cr.store.queue.replicate.purgatory", title: "Replicas in Purgatory" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.split.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.split.pending", title: "Pending Actions" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.gc.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.gc.pending", title: "Pending Actions" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.raftlog.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.raftlog.pending", title: "Pending Actions" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.consistency.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.consistency.pending", title: "Pending Actions" },
      ],
    },
    {
      type: "metrics",
      measure: "count",
      metrics: [
        { name: "cr.store.queue.tsmaintenance.process.success", title: "Successful Actions / sec" },
        { name: "cr.store.queue.tsmaintenance.pending", title: "Pending Actions" },
      ],
    },
  ],
};

export default function (props: GraphDashboardProps) {
  const { storeSources } = props;

  return [
    <LineGraph title="Queue Processing Failures" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.gc.process.failure" title="GC" nonNegativeRate />
        <Metric name="cr.store.queue.replicagc.process.failure" title="Replica GC" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.process.failure" title="Replication" nonNegativeRate />
        <Metric name="cr.store.queue.split.process.failure" title="Split" nonNegativeRate />
        <Metric name="cr.store.queue.consistency.process.failure" title="Consistency" nonNegativeRate />
        <Metric name="cr.store.queue.raftlog.process.failure" title="Raft Log" nonNegativeRate />
        <Metric name="cr.store.queue.tsmaintenance.process.failure" title="Time Series Maintenance" nonNegativeRate />
      </Axis>
    </LineGraph>,

    <LineGraph title="Queue Processing Times" sources={storeSources}>
      <Axis units={AxisUnits.Duration}>
        <Metric name="cr.store.queue.gc.processingnanos" title="GC" nonNegativeRate />
        <Metric name="cr.store.queue.replicagc.processingnanos" title="Replica GC" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.processingnanos" title="Replication" nonNegativeRate />
        <Metric name="cr.store.queue.split.processingnanos" title="Split" nonNegativeRate />
        <Metric name="cr.store.queue.consistency.processingnanos" title="Consistency" nonNegativeRate />
        <Metric name="cr.store.queue.raftlog.processingnanos" title="Raft Log" nonNegativeRate />
        <Metric name="cr.store.queue.tsmaintenance.processingnanos" title="Time Series Maintenance" nonNegativeRate />
      </Axis>
    </LineGraph>,

    // TODO(mrtracy): The queues below should also have "processing
    // nanos" on the graph, but that has a time unit instead of a count
    // unit, and thus we need support for multi-axis graphs.
    <LineGraph title="Replica GC Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.replicagc.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicagc.pending" title="Pending Actions" downsampleMax />
        <Metric name="cr.store.queue.replicagc.removereplica" title="Replicas Removed / sec" nonNegativeRate />
      </Axis>
    </LineGraph>,

    <LineGraph title="Replication Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.replicate.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.pending" title="Pending Actions" />
        <Metric name="cr.store.queue.replicate.addreplica" title="Replicas Added / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.removereplica" title="Replicas Removed / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.removedeadreplica" title="Dead Replicas Removed / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.rebalancereplica" title="Replicas Rebalanced / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.transferlease" title="Leases Transferred / sec" nonNegativeRate />
        <Metric name="cr.store.queue.replicate.purgatory" title="Replicas in Purgatory" downsampleMax />
      </Axis>
    </LineGraph>,

    <LineGraph title="Split Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.split.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.split.pending" title="Pending Actions" downsampleMax />
      </Axis>
    </LineGraph>,

    <LineGraph title="GC Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.gc.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.gc.pending" title="Pending Actions" downsampleMax />
      </Axis>
    </LineGraph>,

    <LineGraph title="Raft Log Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.raftlog.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.raftlog.pending" title="Pending Actions" downsampleMax />
      </Axis>
    </LineGraph>,

    <LineGraph title="Consistency Checker Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.consistency.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.consistency.pending" title="Pending Actions" downsampleMax />
      </Axis>
    </LineGraph>,

    <LineGraph title="Time Series Maintenance Queue" sources={storeSources}>
      <Axis>
        <Metric name="cr.store.queue.tsmaintenance.process.success" title="Successful Actions / sec" nonNegativeRate />
        <Metric name="cr.store.queue.tsmaintenance.pending" title="Pending Actions" downsampleMax />
      </Axis>
    </LineGraph>,
  ];
}
