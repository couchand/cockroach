import { assert } from "chai";

import { selectLocations, selectLocationTree } from "./locations";

function makeStateWithLocations(locationData) {
  return {
    cachedData: {
      locations: {
        valid: true,
        data: {
          locations: locationData,
        },
      },
    },
  };
}

describe("selectLocations", function() {
  it("returns an empty array if location data is missing", function() {
    const state = {
      cachedData: {
        locations: {},
      },
    };

    assert.deepEqual(selectLocations(state), []);
  });

  it("returns an empty array if location data is invalid", function() {
    const state = {
      cachedData: {
        locations: {
          valid: false,
          data: {},
        },
      },
    };

    assert.deepEqual(selectLocations(state), []);
  });

  it("returns location data if valid", function() {
    const locationData = [1, 2, 3];
    const state = makeStateWithLocations(locationData);

    assert.deepEqual(selectLocations(state), locationData);
  });
});

describe("selectLocationTree", function() {
  it("returns an empty object if locations are empty", function() {
    const state = makeStateWithLocations([]);

    assert.deepEqual(selectLocationTree(state), {});
  });

  it("makes a key for each locality tier in locations", function() {
    const tiers = [
      "region",
      "city",
      "data-center",
      "rack",
    ];
    const locations = tiers.map((tier) => ({ localityKey: tier }));
    const state = makeStateWithLocations(locations);

    assert.hasAllKeys(selectLocationTree(state), tiers);
  });

  it("makes a key for each locality value in each tier", function() {
    const cities = [
      "nyc",
      "sf",
      "walla-walla",
    ];
    const dataCenters = [
      "us-east-1",
      "us-west-1",
    ];
    const cityLocations = cities.map((city) => ({ localityKey: "city", localityValue: city }));
    const dcLocations = dataCenters.map((dc) => ({ localityKey: "data-center", localityValue: dc }));
    const state = makeStateWithLocations(cityLocations.concat(dcLocations));

    const tree = selectLocationTree(state);

    assert.hasAllKeys(tree, ["city", "data-center"]);
    assert.hasAllKeys(tree["city"], cities);
    assert.hasAllKeys(tree["data-center"], dataCenters);
  });

  it("returns each location under its key and value", function() {
    const us = { localityKey: "country", localityValue: "US" };
    const nyc = { localityKey: "city", localityValue: "NYC" };
    const sf = { localityKey: "city", localityValue: "SF" };
    const locations = [us, nyc, sf];
    const state = makeStateWithLocations(locations);

    const tree = selectLocationTree(state);

    assert.equal(tree.country.US, us);
    assert.equal(tree.city.NYC, nyc);
    assert.equal(tree.city.SF, sf);
  });
});
