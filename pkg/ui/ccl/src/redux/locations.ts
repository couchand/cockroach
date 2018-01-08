import d3 from "d3";
import { createSelector } from "reselect";

import { AdminUIState } from "src/redux/state";

export function selectLocations(state: AdminUIState) {
  if (!state.cachedData.locations.valid) return [];

  return state.cachedData.locations.data.locations;
}

const nestLocations = d3.nest()
  .key((loc) => loc.localityKey)
  .key((loc) => loc.localityValue)
  .rollup((locations) => locations[0]) // cannot collide since ^^ is primary key
  .map;

export const selectLocationTree = createSelector(
  selectLocations,
  nestLocations,
}
