import { AxisUnits } from "oss/src/views/shared/components/metricQuery";
import { GraphDashboardProps } from "oss/src/views/cluster/containers/nodeGraphs/dashboards/dashboardUtils";

export type Measure = "count" | "bytes" | "duration";

export type Metric = {
  name: string,
  sources?: string[],
  title?: string,
  rate?: boolean,
  nonNegativeRate?: boolean,
  aggregateMax?: boolean,
  aggregateMin?: boolean,
  aggregateAvg?: boolean,
  downsampleMax?: boolean,
  downsampleMin?: boolean,
};

// Chart representing multiple metrics.
type MetricsChartConfig = {
  metrics: Metric[],
};

// Chart representing a single metric across multiple nodes.
type NodesChartConfig = {
  metric: Metric,
};

export type ChartDataConfig = MetricsChartConfig | NodesChartConfig;

type CommonChartConfig = {
  title: string,
  subtitle?: string,
  measure: Measure,
  sources?: string[],
  tooltip?: string | JSX.Element,
  axis?: AxisConfig,
};

export type AxisConfig = {
  units?: AxisUnits,
  label?: string,
};

export type ChartConfig = CommonChartConfig & ChartDataConfig;

export type DashboardConfig = {
  // Unique identifier for this dashboard, e.g. "nodes.overview"
  id: string,
  charts: ChartConfig[],
};

export type DashboardConfigState = (props: GraphDashboardProps) => DashboardConfig;

// export function isMetricsChart(chart: ChartDataConfig): chart is MetricsChartConfig {
//   return chart.type === "metrics";
// }
//
// export function isNodesChart(chart: ChartDataConfig): chart is NodesChartConfig {
//   return chart.type === "nodes";
// }
