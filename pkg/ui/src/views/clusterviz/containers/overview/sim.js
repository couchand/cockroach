import * as d3 from "d3";
import "d3-path";
import { line, curveCardinalOpen } from "d3-shape";

import { LivenessStatus } from "src/redux/nodes";
import { MetricConstants } from "src/util/proto";

import worldGeoPaths from "./world.json";
import usStatesGeoPaths from "./us-states.json";

d3.line = line;
d3.curveCardinalOpen = curveCardinalOpen;

// Temporary hard-coded locations.
var locations = {
  "city=New York City": [-74.00597, 40.71427],
  "city=Miami": [-80.19366, 25.77427],
  "city=Dallas": [-96.7970, 32.7767],
  "city=Des Moines": [-93.60911, 41.60054],
  "city=Los Angeles": [-118.24368, 34.05223],
  "city=Seattle": [-122.33207, 47.60621],
  "city=London": [-0.12574, 51.50853],
  "city=Berlin": [13.41053, 52.52437],
  "city=Stockholm": [18.0649, 59.33258],
  "city=Sydney": [151.20732, -33.86785],
  "city=Melbourne": [144.96332, -37.814],
  "city=Brisbane": [153.02809, -27.46794],
  "city=Beijing": [116.39723, 39.9075],
  "city=Shanghai": [121.45806, 31.22222],
  "city=Shenzhen": [114.0683, 22.54554],
  "city=Mumbai": [72.88261, 19.07283],
  "city=Bangalore": [77.59369, 12.97194],
  "city=New Delhi": [77.22445, 28.63576],
};

export function initNodeCanvas(svg) {
  var model = new Model(svg, d3.select(svg));

  layoutProjection(model);

  window.onpopstate = function(event) {
    if (event.state == null) {
      model.setLocality([]);
      zoomToLocality(model, 750);
      return;
    }
    model.setLocality(event.state.locality);
    zoomToLocality(model, 750);
  }

  return model;
}

export function updateNodeCanvas(model, nodesSummary) {
  var first = model.nodes.length == 0 && nodesSummary.nodeStatuses.length > 0;
  model.livenesses = nodesSummary.livenessStatusByNodeID;
  model.nodes = nodesSummary.nodeStatuses;
  model.updateNodeHistory(nodesSummary.nodeStatusByID);

  model.resetLocalities();
  if (first) {
    zoomToLocality(model, 0);
  } else {
    model.redraw();
  }
}

/* node */

function nodeID(node) {
  return node.desc.node_id;
}

function nodeState(model, node) {
  if (model.livnesses == null) {
    return LivenessStatus.HEALTHY;
  }
  return model.livenesses[nodeID(node)];
}

function nodeName(node) {
  return node.desc.address.address_field;
}

function nodeLocality(node) {
  var locality = [];
  for (var i = 0; i < node.desc.locality.tiers.length; i++) {
    var tier = node.desc.locality.tiers[i];
    locality.push(tier.key + "=" + tier.value);
  }
  locality.push("node=" + nodeName(node));
  return locality;
}

function nodeLocation(node) {
  var locality = nodeLocality(node);
  for (var i = locality.length - 1; i >= 0; i--) {
    var loc = locality[i];
    if (loc in locations) {
      return locations[loc];
    }
  }
  return [0, 0];
}

function nodeAdjustLocation(node, location) {
  locations["node=" + nodeName(node)] = location;
}

function nodeUsage(node) {
  return node.metrics[MetricConstants.usedCapacity];
}

function nodeCapacity(node) {
  return node.metrics[MetricConstants.availableCapacity];
}

function nodeClientActivity(node, model) {
  var last = model.getLastStatus(nodeID(node), node.updated_at);
  if (last == null) {
    return 0;
  }
  var seconds = node.updated_at.subtract(last.updated_at).divide(1000000).toNumber() / 1000,
      totalQPS = (node.metrics["sql.select.count"] - last.metrics["sql.select.count"]) +
      (node.metrics["sql.distsql.select.count"] - last.metrics["sql.distsql.select.count"]) +
      (node.metrics["sql.update.count"] - last.metrics["sql.update.count"]) +
      (node.metrics["sql.insert.count"] - last.metrics["sql.insert.count"]) +
      (node.metrics["sql.delete.count"] - last.metrics["sql.delete.count"]);
  return totalQPS / seconds;
}

// networkActivity returns a tuple of values: [outgoing throughput,
// incoming throughput, average latency]. Throughput values are in
// bytes / s; latency is in milliseconds.
function nodeNetworkActivity(node, model, filter) {
  var last = model.getLastStatus(nodeID(node), node.updated_at),
      activity = [0, 0, 0],
      count = 0;

  if (last != null) {
    var seconds = node.updated_at.subtract(last.updated_at).divide(1000000).toNumber() / 1000;
    for (var key in node.activity) {
      if (key == nodeID(node)) continue;
      if (filter == null || (key in filter)) {
        if (key in last.activity) {
          activity[0] += (node.activity[key].outgoing.toNumber() - last.activity[key].outgoing.toNumber());
        }
      }
    }
    activity[0] /= seconds;
    for (var key in node.activity) {
      if (key == nodeID(node)) continue;
      if (filter == null || (key in filter)) {
        if (key in last.activity) {
          activity[1] += (node.activity[key].incoming.toNumber() - last.activity[key].incoming.toNumber());
        }
      }
    }
    activity[1] /= seconds;
  }
  for (var key in node.activity) {
    if (key == nodeID(node)) continue;
    if (filter == null || (key in filter)) {
      activity[2] += node.activity[key].latency.divide(1000000).toNumber();
      count++;
    }
  }
  activity[2] /= count;
  return activity;
}

/* model */

function Model(svgParent, svg) {
  this.svgParent = svgParent;
  this.svg = svg;

  this.localityRadius = 42;
  this.nodes = [];
  this.nodeHistory = {};
  this.livenesses = [];
  this.localities = [];
  this.currentLocality = [];
  this.maxClientActivity = 1;
  this.maxNetworkActivity = 1;
  this.projection = d3.geo.mercator();
}

Model.prototype.updateNodeHistory = function(statusByID) {
  var maxHistory = 2; // max 2 historical results
  for (var key in statusByID) {
    if (!(key in this.nodeHistory)) {
      this.nodeHistory[key] = [statusByID[key]];
    } else {
      var history = this.nodeHistory[key];
      if (history[history.length - 1].updated_at.lessThan(statusByID[key].updated_at)) {
        this.nodeHistory[key].push(statusByID[key]);
        if (this.nodeHistory[key].length > maxHistory) {
          this.nodeHistory[key] = this.nodeHistory[key].slice(1);
        }
      }
    }
  }
}

Model.prototype.getLastStatus = function(nodeID, updatedAt) {
  if (!(nodeID in this.nodeHistory)) {
    return null;
  }
  for (var i = this.nodeHistory[nodeID].length - 1; i >= 0; i--) {
    var status = this.nodeHistory[nodeID][i];
    if (status.updated_at.lessThan(updatedAt)) {
      return status;
    }
  }
  return null;
}

Model.prototype.width = function() {
  return this.svgParent.clientWidth;
}

Model.prototype.height = function() {
  return this.svgParent.clientHeight;
}

Model.prototype.maxRadius = function() {
  return this.localityRadius * 1.6;
}

// bounds returns longitude / latitude pairs representing the minimum
// and maximum bounds.
Model.prototype.bounds = function() {
  var locXYMin = [180, -90],
      locXYMax = [-180, 90];

  for (var locality of this.localities) {
    var loc = locality.location;
    if (loc[0] < locXYMin[0]) {
      locXYMin[0] = loc[0];
    }
    if (loc[0] > locXYMax[0]) {
      locXYMax[0] = loc[0];
    }
    if (loc[1] > locXYMin[1]) {
      locXYMin[1] = loc[1];
    }
    if (loc[1] < locXYMax[1]) {
      locXYMax[1] = loc[1];
    }
  }

  return [locXYMin, locXYMax];
}

Model.prototype.setLocality = function(locality) {
  this.currentLocality = locality;
  this.resetLocalities();
}

// localityKey concatenates locality information into a comma-separated
// string for use as the key in a dictionary.
function localityKey(locality) {
  var key = '';
  for (var i = 0; i < locality.length; i++) {
    if (i > 0) {
      key += ",";
    }
    key += (i + ":" + locality[i]);
  }
  return key
}

// localityHasPrefix checks whether the supplied locality array has
// the supplied prefix.
function localityHasPrefix(locality, prefix) {
  if (prefix.length > locality.length) {
    return false;
  }
  for (var i = 0; i < prefix.length; i++) {
    if (locality[i] != prefix[i]) {
      return false;
    }
  }
  return true;
}

Model.prototype.resetLocalities = function() {
  // Determine localities to display based on current locality.
  var localityMap = {};
  this.localities = [];
  this.localityLinks = [];
  this.localityScale = 1;
  this.maxClientActivity = 1;
  this.maxNetworkActivity = 1;
  for (var i = 0; i < this.nodes.length; i++) {
    var node = this.nodes[i],
        nodeLoc = nodeLocality(node);
    if (localityHasPrefix(nodeLoc, this.currentLocality)) {
      var locality = nodeLoc.slice(0, this.currentLocality.length + 1);
      var key = localityKey(locality);
      if (!(key in localityMap)) {
        localityMap[key] = {
          locality: locality,
          nodes: [],
        };
      }
      localityMap[key].nodes.push(node);
    }
  }
  for (var loc in localityMap) {
    var l = new Locality(localityMap[loc].locality, localityMap[loc].nodes, this);
    this.localities.push(l);

    // Initialize the max client and network activity values for the displayed localities.
    l.clientActivity();
    l.totalNetworkActivity();
  }
  for (var l1 of this.localities) {
    for (var l2 of this.localities) {
      if (l1 == l2) continue;
      this.localityLinks.push(new LocalityLink(l1, l2, this));
    }
  }
  layoutModel(this);
}

function distance(n1, n2) {
  return Math.sqrt((n1[0] - n2[0]) * (n1[0] - n2[0]) + (n1[1] - n2[1]) * (n1[1] - n2[1]));
}

function length(v) {
  return Math.sqrt(v[0]*v[0] + v[1]*v[1]);
}

function add(v1, v2) {
  return [v1[0] + v2[0], v1[1] + v2[1]];
}

function sub(v1, v2) {
  return [v1[0] - v2[0], v1[1] - v2[1]];
}

function mult(v1, scalar) {
  return [v1[0] * scalar, v1[1] * scalar];
}

function normalize(v) {
  var l = length(v);
  if (l == 0) {
    return [0, 0];
  }
  return [v[0] / l, v[1] / l];
}

function invert(v) {
  return [v[1], -v[0]];
}

function dotprod(v1, v2) {
  return v1[0] * v2[0] + v1[1] * v2[1];
}

// computeLocalityScale returns a scale factor in the interval (0, 1]
// that allows for all localities to exist with no overlap.
// TODO(spencer): use a quadtree for performance if there are many
// localities.
Model.prototype.computeLocalityScale = function() {
  var scale = 1,
      maxDistance = this.maxRadius() * 2;
  for (var i = 0; i < this.localities.length; i++) {
    this.localities[i].pos = this.projection(this.localities[i].location);
  }

  for (var i = 0; i < this.localityLinks.length; i++) {
    var link = this.localityLinks[i],
        d = distance(link.l1.pos, link.l2.pos);
    if (d < maxDistance) {
      var newScale = d / maxDistance;
      if (newScale < scale) {
        scale = newScale;
      }
    }
  }
  this.localityScale = scale;
}

// findClosestPoint locates the closest point on the vector starting
// from point s and extending through u (t=1), nearest to point p.
// Returns an empty vector if the closest point is either start or end
// point or located before or after the line segment defined by [s,
// e].
function findClosestPoint(s, e, p) {
  // u = e - s
  // v = s+tu - p
  // d = length(v)
  // d = length((s-p) + tu)
  // d = sqrt(([s-p].x + tu.x)^2 + ([s-p].y + tu.y)^2)
  // d = sqrt([s-p].x^2 + 2[s-p].x*tu.x + t^2u.x^2 + [s-p].y^2 + 2[s-p].y*tu.y + t^2*u.y^2)
  // ...minimize with first derivative with respect to t
  // 0 = 2[s-p].x*u.x + 2tu.x^2 + 2[s-p].y*u.y + 2tu.y^2
  // 0 = [s-p].x*u.x + tu.x^2 + [s-p].y*u.y + tu.y^2
  // t*(u.x^2 + u.y^2) = [s-p].x*u.x + [s-p].y*u.y
  // t = ([s-p].x*u.x + [s-p].y*u.y) / (u.x^2 + u.y^2)
  var u = sub(e, s),
      d = sub(s, p),
      t = -(d[0]*u[0] + d[1]*u[1]) / (u[0]*u[0] + u[1]*u[1]);
  if (t <= 0 || t >= 1) {
    return [0, 0];
  }
  return add(s, mult(u, t));
}

// computeLocalityLinkPaths starts with a line between the outer radii
// of each locality. Each line is drawn as a cardinal curve. This
// initially straight curve is intersected with each locality and bent
// in order to avoid intersection. The bending is straightforward and
// will by no means avoid intersections entirely.
Model.prototype.computeLocalityLinkPaths = function() {
  var maxR = this.localityRadius * 1.11111 * this.localityScale;
  for (var i = 0; i < this.localityLinks.length; i++) {
    var link = this.localityLinks[i];
    // If the link goes from right to left, mark it as reversed.
    if (link.l1.pos[0] > link.l2.pos[0]) {
      link.reversed = true
    }
    var vec = sub(link.l2.pos, link.l1.pos),
        len = length(vec),
        norm = normalize(vec),
        skip = maxR;

    // Create the first points in the curve between the two localties.
    link.points = [link.l1.pos, add(link.l1.pos, mult(norm, skip))];

    // Bend the curve around any localities which come too close to
    // the line drawn to represent this locality link. This inner
    // loop just adds additional points to the cardinal curve.
    var additionalPoints = [];
    for (var j = 0; j < this.localities.length; j++) {
      // First, find the closest point on the locality link segment to
      // the center of each locality.
      var loc = this.localities[j],
          closest = findClosestPoint(link.l1.pos, link.l2.pos, loc.pos);
      // Only consider bending the locality link IF the closest point
      // lies on the segment _between_ the two localities.
      if (closest != [0, 0]) {
        // We bend if the distance is within 2x the max radius (2x is
        // determined empirically for aesthetics).
        var dist = distance(closest, loc.pos);
        if (dist < maxR * 2) {
          // This part is a bit dicey, so here's an explanation of the
          // algorithm:
          // - Compute the vector from the locality center to closest point.
          // - Measure the angle; if between 45 degrees and 135 degrees:
          //   - If vector points down, bend 2x the max radius to clear the
          //     locality name tag.
          //   - Otherwise, bend 1.5x max radius.
          var cVec = sub(closest, loc.pos),
              angle = (cVec[0] == 0) ? Math.PI / 2 : Math.abs(Math.atan(cVec[1] / cVec[0])),
              magnitude = (angle < Math.PI * 3 / 4 && angle > Math.PI / 4) ? (cVec[1] > 1 ? maxR * 2 : maxR * 1.5) : maxR * 1.5,
              invertNorm = invert(norm),
              perpV = mult(invertNorm, magnitude),
              dir1 = add(loc.pos, perpV),
              dir2 = sub(loc.pos, perpV);
          if (dist < magnitude) {
            if (distance(closest, dir1) < distance(closest, dir2)) {
              link.points.push(dir1);
            } else {
              link.points.push(dir2);
            }
          }
        }
      }
    }

    // Finish up the curve by adding the final points.
    link.points.push(sub(link.l2.pos, mult(norm, skip)));
    link.points.push(link.l2.pos);

    // Ensure that points sort from left to right.
    link.points.sort(function(a, b) { return a[0] - b[0]; });
  }
}

// adjustLocation adjusts the locality location so that it lies on a
// circle of size radius.
Model.prototype.adjustLocations = function(radius) {
  for (var i = 0; i < this.localities.length; i++) {
    var angle = 2 * Math.PI * i / this.localities.length - Math.PI / 2,
        xy = this.projection(this.localities[i].location),
        xyAdjusted = [xy[0] + radius * Math.cos(angle), xy[1] + radius * Math.sin(angle)],
        loc = this.projection.invert(xyAdjusted);
    this.localities[i].location = loc;
    for (var n of this.localities[i].nodes) {
      nodeAdjustLocation(n, loc);
    }
  }
}

/* locality.js */

function Locality(locality, nodes, model) {
  this.locality = locality;
  this.nodes = nodes;
  this.model = model;
  this.location = this.findCentroid();
}

// localityName extracts the locality name as the first element of the
// locality array and strips out any leading ".*=" pattern.
Locality.prototype.name = function() {
  if (this.locality.length == 0) {
    return "";
  }
  var name = this.locality[this.locality.length - 1],
      idx = name.indexOf("=");
  if (idx != -1) {
    return name.substr(idx + 1, name.length);
  }
  return name;
}

Locality.prototype.findCentroid = function() {
  var centroid = [0, 0];
  for (var i = 0; i < this.nodes.length; i++) {
    centroid = [centroid[0] + nodeLocation(this.nodes[i])[0],
                centroid[1] + nodeLocation(this.nodes[i])[1]];
  }
  return [centroid[0] / this.nodes.length, centroid[1] / this.nodes.length];
}

Locality.prototype.state = function() {
  var liveCount = this.liveCount();
  if (liveCount == 0) {
    return "unavailable";
  } else if (liveCount < this.nodes.length) {
    return "mixed";
  }
  return "available";
}

Locality.prototype.liveCount = function() {
  var count = 0;
  for (var i = 0; i < this.nodes.length; i++) {
    count += (nodeState(this.nodes[i]) == LivenessStatus.HEALTHY) ? 1 : 0;
  }
  return count;
}

Locality.prototype.usage = function() {
  var usage = 0;
  for (var i = 0; i < this.nodes.length; i++) {
    usage += nodeUsage(this.nodes[i]);
  }
  return usage;
}

Locality.prototype.capacity = function() {
  var capacity = 0;
  for (var i = 0; i < this.nodes.length; i++) {
    capacity += nodeCapacity(this.nodes[i]);
  }
  return capacity;
}

Locality.prototype.clientActivity = function() {
  var activity = 0;
  for (var i = 0; i < this.nodes.length; i++) {
    activity += nodeClientActivity(this.nodes[i], this.model);
  }
  if (activity > this.model.maxClientActivity) {
    this.model.maxClientActivity = activity;
  }
  return activity;
}

Locality.prototype.totalNetworkActivity = function() {
  var total = [0, 0];
  for (var n of this.nodes) {
    var activity = nodeNetworkActivity(n, this.model, null);
    total[0] += activity[0];
    total[1] += activity[1];
  }
  var activity = total[0] + total[1];
  if (activity > this.model.maxNetworkActivity) {
    this.model.maxNetworkActivity = activity;
  }
  return activity;
}

/* locality_link.js */

function LocalityLink(l1, l2, model) {
  this.id = l1.name() + "-" + l2.name();
  this.l1 = l1;
  this.l2 = l2;
  this.reversed = false;
  this.model = model;
}

// networkActivity returns a tuple of values: [outgoing throughput,
// incoming throughput, average latency].
LocalityLink.prototype.networkActivity = function() {
  var filter = {};
  for (var n2 of this.l2.nodes) {
    filter[nodeID(n2)] = null;
  }
  var total = [0, 0, 0],
      count = 0;
  for (var n1 of this.l1.nodes) {
    var activity = nodeNetworkActivity(n1, this.model, filter);
    total[0] += activity[0];
    total[1] += activity[1];
    total[2] += activity[2];
    count++;
  }
  total[2] /= count;
  return total;
}

/* visualization.js */

function layoutProjection(model) {
  model.projectionG = model.svg.append("g");

  model.worldG = model.projectionG.append("g");
  model.worldG.selectAll("path")
    .data(worldGeoPaths.features)
    .enter().append("path")
    .attr("class", "geopath")
    .attr("id", function(d) { return "world-" + d.id; });

  model.usStatesG = model.projectionG.append("g");
  model.usStatesG.selectAll("path")
    .data(usStatesGeoPaths.features)
    .enter().append("path")
    .attr("class", "geopath");

  var pathGen = d3.geo.path().projection(model.projection);

  // Compute the scale intent (min to max zoom).
  var minScale = model.width() / 2 / Math.PI,
      maxScale = maxScaleFactor * minScale,
      scaleExtent = [minScale, maxScale * 1024];

  model.maxScale = maxScale;
  model.zoom = d3.behavior.zoom()
    .scaleExtent(scaleExtent)
    .on("zoom", function() {
      // Instead of translating the projection, rotate it (compute yaw as longitudinal rotation).
      var t = model.zoom.translate(),
          s = model.zoom.scale();

      // Compute limits for horizontal and vertical translation based on max latitude.
      model.projection.scale(s).translate([0, 0]);
      var p = model.projection([180, maxLatitude]);
      if (t[0] > p[0]) {
        t[0] = p[0];
      } else if (t[0] + p[0] < model.width()) {
        t[0] = model.width() - p[0];
      }
      if (t[1] > -p[1]) {
        t[1] = -p[1];
      } else if (t[1] - p[1] < model.height()) {
        t[1] = model.height() + p[1];
      }
      if (t[0] != model.zoom.translate()[0] || t[1] != model.zoom.translate()[1]) {
        model.zoom.translate(t);
      }
      model.projection
        .translate(t)
        .scale(s);

      model.projectionG.selectAll("path").attr("d", pathGen);

      // Draw US states if they intersect our viewable area.
      var usB = [model.projection(usStatesBounds[0]), model.projection(usStatesBounds[1])];
      var usScale = (usB[1][1] - usB[0][1]) / model.width();
      if (usB[0][0] < model.width() && usB[1][0] > 0 && usB[0][1] < model.height() && usB[1][1] > 0 && usScale >= 0.2) {
        // Set opacity based on zoom scale.
        var opacity = (usScale - 0.2) / (0.33333 - 0.2)
        model.usStatesG.style("opacity",  opacity);
        // Set opacity for the considerably less detailed world map's version of the US.
        model.worldG.select("#world-840").style("opacity", opacity < 1 ? 1 : 0);
      } else {
        model.usStatesG.style("opacity", 0);
        model.worldG.select("#world-840").style("opacity", 1);
      }
      // Fade out geographic projection when approaching max scale.
      model.projectionG.style("opacity", 1 - Math.min(1, (s / maxScale)));

      if (model.redraw != null) {
        model.redraw();
      }
    })
    .on("zoomend", function() {
      model.svg.call(model.zoom); // enable manual pan and zoom
    });

  model.projection.scale(model.maxScale);
}

var usStatesBounds = [[-124.626080, 48.987386], [-62.361014, 18.005611]],
    maxLatitude = 83, // clip northern and southern poles
    maxScaleFactor = 100;

function findScale(b1, b2, factor) {
  if (b1 == b2) {
    return 0.0;
  }
  return factor / Math.abs(b2 - b1) / 1.5;
}

function zoomToLocality(model, duration, updateHistory) {
  // While zooming into a locality, disable manual pan & zoom.
  model.svg.on('.zoom', null);

  var bounds = model.bounds(),
      scalex = findScale(bounds[0][0], bounds[1][0], model.width() / (Math.PI / 180)),
      scaley = findScale(bounds[0][1], bounds[1][1], model.height() / (Math.PI / 90)),
      scale = scalex == 0 ? scaley : (scaley == 0 ? scalex : Math.min(scalex, scaley)),
      needAdjust = false;

  if (scale == 0) {
    needAdjust = true;
    scale = model.maxScale * Math.pow(2, model.currentLocality.length);
  }

  // Compute the initial translation to center the deployed datacenters.
  model.projection.rotate([0, 0]).scale(scale).translate([0, 0]);
  var center = [(bounds[0][0] + bounds[1][0]) / 2, (bounds[0][1] + bounds[1][1]) / 2];
  var p = model.projection(center);

  // If necessary (all localities had the same location), adjust the
  // location of each locality so there's some differentiation for
  // display purposes.
  if (needAdjust) {
    model.adjustLocations(0.3 * Math.min(model.width(), model.height()));
    bounds = model.bounds();
  }

  model.svg
    .transition()
    .duration(duration)
    .call(model.zoom
          .translate([model.width() / 2 - p[0], model.height() / 2 - p[1]])
          .scale(scale)
          .event);

  if (updateHistory) {
    history.pushState({modelID: model.id, locality: model.currentLocality.slice(0)}, "");
  }
}

// createArcPath returns an svg arc object. startAngle and endAngle are
// expressed in radians.
function createArcPath(innerR, outerR, startAngle, endAngle) {
  return d3.svg.arc()
    .innerRadius(innerR)
    .outerRadius(outerR)
    .startAngle(startAngle)
    .endAngle(endAngle)()
}

function drawBox(w, h, cornerPct) {
  var c = w * cornerPct;
  return "M" + c + ",0 L" + (w-c) + ",0 A" + c + "," + c + " 0 0 1 " + w + "," + c +
    " L" + w + "," + (h-c) + " A" + c + "," + c + " 0 0 1 " + (w-c) + "," + h +
    " L" + c + "," + h + " A" + c + "," + c + " 0 0 1 0," + (h-c) +
    " L0," + c + " A" + c + "," + c + " 0 0 1 " + c + ",0 Z";
}

function arcAngleFromPct(pct) {
  return Math.PI * (pct * 1.25 - 0.75);
}

function angleFromPct(pct) {
  return Math.PI * (-1.25 + 1.25 * pct);
}

function bytesToSize(bytes) {
  var sizes = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  if (bytes == 0) return '0 B';
  var i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
  return Math.round(bytes * 10 / Math.pow(1024, i), 2) / 10 + ' ' + sizes[i];
}

function bytesToActivity(bytes) {
  var sizes = ['B/s', 'KiB/s', 'MiB/s', 'GiB/s', 'TiB/s'];
  if (bytes < 1) return '0 B/s';
  var i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
  return Math.round(bytes * 10 / Math.pow(1024, i), 2) / 10 + ' ' + sizes[i];
}

function latencyMilliseconds(latency) {
  return Math.round(latency) + ' ms';
}

function showLocalityLinks(model, locality) {
  model.svg.selectAll(".locality-link-group")
    .transition()
    .duration(250)
    .attr("visibility", function(d) { return d.l1.name() == locality.name() ? "visible" : "hidden"; })
    .attr("opacity", function(d) { return d.l1.name() == locality.name() ? 1 : 0; });
}

function hideLocalityLinks(model, locality) {
  model.svg.selectAll(".locality-link-group")
    .transition()
    .duration(250)
    .attr("visibility", "hidden")
    .attr("opacity", 0);
}

function layoutModel(model) {
  // Locality Links
  var linkSel = model.svg.selectAll(".locality-link-group")
    .data(model.localityLinks, function(d) { return d.id; });
  linkSel.exit().remove();
  var newLinkSel = linkSel.enter().append("g")
      .attr("class", "locality-link-group")
      .attr("opacity", 0)
      .attr("id", function(d) { return d.id; });

  newLinkSel.append("path")
    .attr("id", function(d) { return d.id + "-path"; })
    .attr("class", "locality-link");

  newLinkSel.append("text")
    .attr("id", function(d) { return "incoming-" + d.id; })
    .append("textPath")
    .attr("class", "incoming-throughput-label")
    .attr("startOffset", "50%")
    .attr("xlink:href", function(d) { return "#" + d.id + "-path"; });
  newLinkSel.append("text")
    .attr("id", function(d) { return "outgoing-" + d.id; })
    .append("textPath")
    .attr("class", "outgoing-throughput-label")
    .attr("startOffset", "50%")
    .attr("xlink:href", function(d) { return "#" + d.id + "-path"; })
  newLinkSel.append("text")
    .attr("id", function(d) { return "rtt-" + d.id; })
    .append("textPath")
    .attr("class", "rtt-label")
    .attr("startOffset", "60%")
    .attr("xlink:href", function(d) { return "#" + d.id + "-path"; })

  newLinkSel.selectAll(".locality-link-group")
    .append("use")
    .attr("id", function(d) { return "incoming-" + d.id + "-path"; })
    .attr("xlink:href", function(d) { return "#incoming-" + d.id; });
  newLinkSel.selectAll(".locality-link-group")
    .append("use")
    .attr("id", function(d) { return "outgoing-" + d.id + "-path"; })
    .attr("xlink:href", function(d) { return "#outgoing-" + d.id; });
  newLinkSel.selectAll(".locality-link-group")
    .append("use")
    .attr("id", function(d) { return "rtt-" + d.id + "-path"; })
    .attr("xlink:href", function(d) { return "#rtt-" + d.id; });

  var locSel = model.svg.selectAll(".locality")
      .data(model.localities, function(d) { return d.name(); });
  locSel.exit()
    .transition()
    .duration(500)
    .style("opacity", 0)
    .attr("class", "removed-locality")
    .remove();
  var newLocSel = locSel.enter().append("g")
      .attr("id", function(d) { return d.name(); })
      .attr("class", "locality")
      .style("opacity", 0)
      .on("click", function(d) {
        if (d.nodes.length > 1 && d.locality != model.currentLocality) {
          hideLocalityLinks(model, d);
          model.setLocality(d.locality);
          zoomToLocality(model, 750, true);
        }
      });
  newLocSel
    .transition()
    .duration(750)
    .style("opacity", 1);

  var innerR = model.localityRadius,
      arcWidth = model.localityRadius * 0.11111,
      outerR = innerR + arcWidth;

  // Capacity arc.
  var capacityG = newLocSel.append("g")
      .attr("class", "capacity-centric");

  capacityG.append("path")
    .attr("d", function(d) { return createArcPath(innerR, outerR, arcAngleFromPct(0), arcAngleFromPct(1)); })
    .attr("class", "capacity-background");
  capacityG.append("text")
    .attr("class", "capacity-label");

  // Used capacity arc.
  var usedG = capacityG.append("g");
  usedG.append("path")
    .attr("class", "capacity-used");
  usedG.append("text")
    .attr("class", "capacity-used-label");

  // Capacity labels.
  var capacityLabels = capacityG.append("g")
      .attr("transform", "translate(" + -outerR + ", " + -outerR + ")");
  var capacityLabelsSVG = capacityLabels.append("svg")
      .attr("width", outerR * 2)
      .attr("height", outerR * 2);
  capacityLabelsSVG.append("text")
    .attr("class", "capacity-used-pct-label")
    .attr("x", "50%")
    .attr("y", "40%");
  capacityLabelsSVG.append("text")
    .attr("class", "capacity-used-text")
    .attr("x", "50%")
    .attr("y", "60%")
    .text("CAPACITY USED");

  // Client / network activity.
  var activityG = capacityG.append("g")
      .attr("transform", "translate(" + 0 + ", " + (innerR * Math.sin(angleFromPct(0))) + ")");
  activityG.append("line")
    .attr("class", "client-activity");
  activityG.append("text")
    .attr("class", "client-activity-label");
  activityG.append("line")
    .attr("class", "network-activity");
  activityG.append("text")
    .attr("class", "network-activity-label");

  // Locality label.
  var localityLabels = capacityG.append("g")
      .attr("transform", "translate(" + -outerR + ", " + outerR * 0.9 + ")");
  localityLabels.append("path")
    .attr("d", function(d) { return drawBox(outerR * 2, 20, 0.05); })
    .attr("class", "locality-label-background")
  localityLabels.append("svg")
    .attr("width", function(d) { return outerR * 2 })
    .attr("height", "20")
    .append("text")
    .attr("class", "locality-label")
    .attr("x", "50%")
    .attr("y", "55%")
    .text(function(d) { return d.name(); });

  // Circle for showing inter-locality network links.
  capacityG.append("circle")
    .style("opacity", 0)
    .attr("r", innerR - arcWidth * 2)
    .style("cursor", "pointer")
    .on("mouseover", function(d) { showLocalityLinks(model, d); })
    .on("mouseout", function(d) { hideLocalityLinks(model, d); });

  model.redraw = function() {
    // Now that we've set the projection and adjusted locality locations
    // in the event there are no differences in location, we can compute
    // the factor we need to scale each locality so that they don't
    // overlap.
    model.computeLocalityScale();

    // Compute locality link paths.
    model.computeLocalityLinkPaths();

    model.svg.selectAll(".removed-locality")
      .attr("transform", function(d) {
        return "translate(" + model.projection(d.location) + ")";
      });

    var locSel = model.svg.selectAll(".locality"),
        linkSel = model.svg.selectAll(".locality-link-group");

    locSel.attr("transform", function(d) {
      return "translate(" + d.pos + ")scale(" + model.localityScale + ")";
    });
    linkSel.select(".locality-link")
      .attr("d", function(d) { return d3.line().curve(d3.curveCardinalOpen.tension(0.5))(d.points); });

    updateModel(model, locSel, linkSel);
  }

  model.redraw();
}

function updateModel(model, locSel, linkSel) {
  var innerR = model.localityRadius,
      arcWidth = model.localityRadius * 0.11111,
      outerR = innerR + arcWidth;

  locSel.select(".capacity-label")
    .attr("x", (outerR + arcWidth) * Math.cos(0))
    .text(function(d) { return bytesToSize(d.capacity()); });

  locSel.select(".capacity-used")
    .attr("d", function(d) {
      var pct = d.usage() / d.capacity();
      return createArcPath(innerR, outerR, arcAngleFromPct(0), arcAngleFromPct(pct));
    });

  locSel.select(".capacity-used-label")
    .attr("transform", function(d) {
      var pct = d.usage() / d.capacity(),
          x = Math.cos(angleFromPct(pct)),
          y = Math.sin(angleFromPct(pct)),
          radius = outerR + arcWidth;
      return "translate(" + (x * radius) + "," + (y * radius) + ")";
    })
    .attr("text-anchor", function(d) {
      var pct = d.usage() / d.capacity();
      return (pct < 0.75) ? "end" : "start";
    })
    .text(function(d) { return bytesToSize(d.usage()); });
  locSel.select(".capacity-used-pct-label")
    .text(function(d) {
      var pctUsed = d.usage() / d.capacity();
      if (pctUsed < 0.01) {
        return "." + d3.format("02d")(Math.round(10000 * pctUsed)) + "%";
      } else if (pctUsed < 0.1) {
        return (Math.round(1000 * pctUsed) / 10) + "%";
      }
      return Math.round(100 * pctUsed) + "%";
    });

  var barsX = innerR * Math.cos(angleFromPct(0)),
      barsWidth = outerR - barsX - 4,
      labelX = outerR + arcWidth,
      labelH = 10;
  locSel.select(".client-activity")
    .transition()
    .duration(250)
    .attr("x1", outerR - 2)
    .attr("y1", -labelH)
    .attr("x2", function(d) { return Math.round(outerR - barsWidth * (d.clientActivity() / model.maxClientActivity)); })
    .attr("y2", -labelH);
  locSel.select(".client-activity-label")
    .attr("x", labelX)
    .attr("y", -labelH)
    .text(function(d) { return d3.format(",d")(Math.round(d.clientActivity())) + " qps"; });
  locSel.select(".network-activity")
    .transition()
    .duration(250)
    .attr("x1", outerR - 2)
    .attr("y1", 0)
    .attr("x2", function(d) { return Math.round(outerR - barsWidth * (d.totalNetworkActivity() / model.maxNetworkActivity)); })
    .attr("y2", 0);
  locSel.select(".network-activity-label")
    .attr("x", labelX)
    .attr("y", 0)
    .text(function(d) { return bytesToActivity(d.totalNetworkActivity()); });

  linkSel.select(".incoming-throughput-label")
    .text(function(d) {
      if (d.reversed) {
        return bytesToActivity(d.networkActivity(null)[1]) + "→";
      }
      return "←" + bytesToActivity(d.networkActivity(null)[1]);
    });
  linkSel.select(".outgoing-throughput-label")
    .text(function(d) {
      if (d.reversed) {
        return "←" + bytesToActivity(d.networkActivity(null)[0]);
      }
      return bytesToActivity(d.networkActivity(null)[0]) + "→";
    });
  linkSel.select(".rtt-label")
    .text(function(d) { return latencyMilliseconds(d.networkActivity(null)[2]); });
}
