import React from "react";

import { AxisUnits } from "src/views/shared/components/metricQuery";
import { DashboardConfigState } from "src/util/charts";

import { nodeAddress } from "./dashboardUtils";

const charts: DashboardConfigState = ({ nodeIDs, nodesSummary, nodeSources, tooltipSelection }) => ({
  id: "nodes.sql",
  name: "SQL",
  charts: [
    {
      title: "SQL Connections",
      sources: nodeSources,
      tooltip: `The total number of active SQL connections ${tooltipSelection}.`,
      measure: "count",
      metrics: [
        { title: "Client Connections", name: "cr.node.sql.conns" },
      ],
    },
    {
      title: "SQL Byte Traffic",
      sources: nodeSources,
      tooltip: `The total amount of SQL client network traffic in bytes per second ${tooltipSelection}.`,
      axis: { units: AxisUnits.Bytes, label: "byte traffic" },
      measure: "count",
      metrics: [
        { name: "cr.node.sql.bytesin", title: "Bytes In", nonNegativeRate: true },
        { name: "cr.node.sql.bytesout", title: "Bytes Out", nonNegativeRate: true },
      ],
    },
    {
      title: "SQL Queries",
      sources: nodeSources,
      tooltip: `A ten-second moving average of the # of SELECT, INSERT, UPDATE, and DELETE operations
        started per second ${tooltipSelection}.`,
      measure: "count",
      metrics: [
        { name: "cr.node.sql.select.count", title: "Total Reads" },
        { name: "cr.node.sql.distsql.select.count", title: "DistSQL Reads" },
        { name: "cr.node.sql.update.count", title: "Updates" },
        { name: "cr.node.sql.insert.count", title: "Inserts" },
        { name: "cr.node.sql.delete.count", title: "Deletes" },
      ],
    },
    {
      title: "Active Distributed SQL Queries",
      sources: nodeSources,
      tooltip: `The total number of distributed SQL queries currently running ${tooltipSelection}.`,
      measure: "count",
      metrics: [
        { name: "cr.node.sql.distsql.queries.active", title: "Active Queries" },
      ],
    },
    {
      title: "Active Flows for Distributed SQL Queries",
      tooltip: "The number of flows on each node contributing to currently running distributed SQL queries.",
      measure: "count",
      metrics: nodeIDs.map((node) => (
        { name: "cr.node.sql.distsql.flows.active", title: nodeAddress(nodesSummary, node), sources: [node] }
      )),
    },
    // TODO(vilterp): maybe we should be able to just say that this is a node chart
    // instead of explicitly mapping over the list of node ids here
    {
      title: "Service Latency: SQL, 99th percentile",
      tooltip: (
        <div>
          Over the last minute, this node executed 99% of queries within this time.&nbsp;
          <em>This time does not include network latency between the node and client.</em>
        </div>
      ),
      axis: { units: AxisUnits.Duration },
      measure: "duration", // TODO(vilterp): figure out how to dedup measure vs units
      metrics: nodeIDs.map((node) => ({
        name: "cr.node.sql.service.latency-p99",
        title: nodeAddress(nodesSummary, node),
        sources: [node],
        downsampleMax: true,
      })),
    },
    {
      title: "Service Latency: SQL, 90th percentile",
      tooltip: (
        <div>
          Over the last minute, this node executed 90% of queries within this time.&nbsp;
          <em>This time does not include network latency between the node and client.</em>
        </div>
      ),
      axis: { units: AxisUnits.Duration },
      measure: "duration", // TODO(vilterp): figure out how to dedup measure vs units
      metrics: nodeIDs.map((node) => ({
        name: "cr.node.sql.service.latency-p90",
        title: nodeAddress(nodesSummary, node),
        sources: [node],
        downsampleMax: true,
      })),
    },
    {
      title: "Service Latency: DistSQL, 99th percentile",
      tooltip: `The latency of distributed SQL statements serviced over
                  10 second periods ${tooltipSelection}.`,
      axis: { units: AxisUnits.Duration },
      measure: "duration",
      metrics: nodeIDs.map((node) => ({
        name: "cr.node.sql.distsql.service.latency-p99",
        title: nodeAddress(nodesSummary, node),
        sources: [node],
        downsampleMax: true,
      })),
    },
    {
      title: "Service Latency: DistSQL, 90th percentile",
      tooltip: `The latency of distributed SQL statements serviced over
                  10 second periods ${tooltipSelection}.`,
      axis: { units: AxisUnits.Duration },
      measure: "duration",
      metrics: nodeIDs.map((node) => ({
        name: "cr.node.sql.distsql.service.latency-p90",
        title: nodeAddress(nodesSummary, node),
        sources: [node],
        downsampleMax: true,
      })),
    },
    {
      title: "Execution Latency: 99th percentile",
      tooltip: `The 99th percentile of latency between query requests and responses over a
          1 minute period. Values are displayed individually for each node on each node.`,
      axis: { units: AxisUnits.Duration },
      measure: "duration",
      metrics: nodeIDs.map((node) => ({
        name: "cr.node.exec.latency-p99",
        title: nodeAddress(nodesSummary, node),
        sources: [node],
        downsampleMax: true,
      })),
    },
    {
      title: "Execution Latency: 90th percentile",
      tooltip: `The 90th percentile of latency between query requests and responses over a
          1 minute period. Values are displayed individually for each node on each node.`,
      axis: { units: AxisUnits.Duration },
      measure: "duration",
      metrics: nodeIDs.map((node) => ({
        name: "cr.node.exec.latency-p90",
        title: nodeAddress(nodesSummary, node),
        sources: [node],
        downsampleMax: true,
      })),
    },
    {
      title: "Transactions",
      sources: nodeSources,
      tooltip: `The total number of transactions opened, committed, rolled back,
                  or aborted per second ${tooltipSelection}.`,
      measure: "count",
      metrics: [
        { name: "cr.node.sql.txn.begin.count", title: "Begin" },
        { name: "cr.node.sql.txn.commit.count", title: "Commits" },
        { name: "cr.node.sql.txn.rollback.count", title: "Rollbacks" },
        { name: "cr.node.sql.txn.abort.count", title: "Aborts" },
      ],
    },
    {
      title: "Schema Changes",
      sources: nodeSources,
      tooltip: `The total number of DDL statements per second ${tooltipSelection}.`,
      measure: "count",
      metrics: [
        { name: "cr.node.sql.ddl.count", title: "DDL Statements" },
      ],
    },
  ],
});

export default charts;
