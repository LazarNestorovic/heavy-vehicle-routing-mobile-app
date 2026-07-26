package reststop

import "testing"

// This is the real production data file (backend/data/), not a package-local
// testdata/ fixture - it's used by the running worker too, not just by tests.
const fixture = "../../data/serbia-rest-stops.osm"

func TestLoad(t *testing.T) {
	stops, err := Load(fixture)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(stops) < 1000 {
		t.Fatalf("expected >1000 rest stops for all of Serbia, got %d", len(stops))
	}

	var fuel, parking int
	for _, s := range stops {
		switch s.Amenity {
		case "fuel":
			fuel++
		case "parking":
			parking++
		}
	}
	if fuel == 0 || parking == 0 {
		t.Errorf("expected both fuel and parking stops, got fuel=%d parking=%d", fuel, parking)
	}
}

func TestFinder_Nearest(t *testing.T) {
	stops, err := Load(fixture)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	finder := NewFinder(stops)

	// Somewhere along the Belgrade-Novi Sad highway - should have a fuel/parking
	// stop within a few km, this corridor is one of the busiest in Serbia.
	stop, distanceM, found := finder.Nearest(44.95, 20.25)
	if !found {
		t.Fatal("expected a stop to be found")
	}
	if distanceM > 15000 {
		t.Errorf("nearest stop to a major highway point is implausibly far: %.1fkm (%s, id=%d)", distanceM/1000, stop.Amenity, stop.ID)
	}
	t.Logf("nearest stop: %s %q at %.0fm", stop.Amenity, stop.Name, distanceM)
}

func TestFinder_Nearest_EmptyFinder(t *testing.T) {
	finder := NewFinder(nil)
	_, _, found := finder.Nearest(44.8, 20.4)
	if found {
		t.Error("expected no stop found for an empty finder")
	}
}

func TestFinder_NearestPreferred_BrandMatch(t *testing.T) {
	stops, err := Load(fixture)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	finder := NewFinder(stops)

	// Real "НИС Петрол" station at ~45.2687,19.8115 (near Novi Sad) confirmed
	// present in the data. Query right next to it - the plain Nearest() might
	// return a closer non-NIS station, but NearestPreferred with the brand set
	// must return the NIS one if it's within radius.
	const brand = "НИС Петрол"
	stop, dist, found := finder.NearestPreferred(45.2687, 19.8115, brand, nil, DefaultPreferredRadiusM)
	if !found {
		t.Fatal("expected a preferred stop to be found")
	}
	if stop.Brand != brand {
		t.Errorf("expected brand %q, got %q (dist=%.0fm)", brand, stop.Brand, dist)
	}
}

func TestFinder_NearestPreferred_FavoriteWinsOverBrand(t *testing.T) {
	stops, err := Load(fixture)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	finder := NewFinder(stops)

	favorite := Stop{ID: 999999, Lat: 45.0, Lon: 19.5, Name: "Moja omiljena", Amenity: "fuel"}
	stop, dist, found := finder.NearestPreferred(45.0, 19.5, "НИС Петрол", []Stop{favorite}, DefaultPreferredRadiusM)
	if !found {
		t.Fatal("expected a stop to be found")
	}
	if stop.ID != favorite.ID {
		t.Errorf("expected the exact favorite stop to win (dist should be ~0), got id=%d dist=%.0fm", stop.ID, dist)
	}
}

func TestFinder_NearestPreferred_FallsBackWhenNothingInRadius(t *testing.T) {
	stops, err := Load(fixture)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	finder := NewFinder(stops)

	// A brand that doesn't exist in the data at all - must fall back to plain Nearest.
	stop, _, found := finder.NearestPreferred(44.95, 20.25, "Definitely Not A Real Brand", nil, DefaultPreferredRadiusM)
	if !found {
		t.Fatal("expected fallback to plain Nearest to still find a stop")
	}
	plain, _, _ := finder.Nearest(44.95, 20.25)
	if stop.ID != plain.ID {
		t.Errorf("expected fallback to match plain Nearest (id=%d), got id=%d", plain.ID, stop.ID)
	}
}

func TestFinder_NearestOnRoute_FiltersToCorridor(t *testing.T) {
	// Closer to the target point in a straight line, but nowhere near the
	// route itself (e.g. down a side road into a village).
	offCorridor := Stop{ID: 1, Lat: 45.001, Lon: 20.001, Amenity: "fuel"}
	// Farther from the target point, but sits right on the route.
	onCorridorStop := Stop{ID: 2, Lat: 45.05, Lon: 20.05, Amenity: "fuel"}
	finder := NewFinder([]Stop{offCorridor, onCorridorStop})

	// Sanity check: plain Nearest would pick the off-corridor one.
	if plain, _, _ := finder.Nearest(45.0, 20.0); plain.ID != offCorridor.ID {
		t.Fatalf("test setup broken: expected plain Nearest to pick id=%d, got id=%d", offCorridor.ID, plain.ID)
	}

	routePoints := []Point{{Lat: 45.05, Lon: 20.05}}
	stop, _, found := finder.NearestOnRoute(45.0, 20.0, "", nil, DefaultPreferredRadiusM, routePoints, 500, false)
	if !found {
		t.Fatal("expected a stop to be found")
	}
	if stop.ID != onCorridorStop.ID {
		t.Errorf("expected the on-corridor stop (id=%d) to win over the closer off-corridor one, got id=%d", onCorridorStop.ID, stop.ID)
	}
}

func TestFinder_NearestOnRoute_FallsBackWhenCorridorEmpty(t *testing.T) {
	only := Stop{ID: 1, Lat: 45.0, Lon: 20.0, Amenity: "fuel"}
	finder := NewFinder([]Stop{only})
	routePoints := []Point{{Lat: 50.0, Lon: 30.0}} // nowhere near `only`

	stop, _, found := finder.NearestOnRoute(45.0, 20.0, "", nil, DefaultPreferredRadiusM, routePoints, 500, false)
	if !found {
		t.Fatal("expected fallback to plain NearestPreferred to still find a stop when nothing is on the corridor")
	}
	if stop.ID != only.ID {
		t.Errorf("expected fallback to the only stop available (id=%d), got id=%d", only.ID, stop.ID)
	}
}

func TestFinder_NearestOnRoute_HazmatPrefersFuelOverSlightlyCloserParking(t *testing.T) {
	parking := Stop{ID: 1, Lat: 45.0001, Lon: 20.0, Amenity: "parking"} // closest
	fuel := Stop{ID: 2, Lat: 45.02, Lon: 20.0, Amenity: "fuel"}         // ~2.2km farther, within hazmat tolerance
	finder := NewFinder([]Stop{parking, fuel})
	routePoints := []Point{{Lat: 45.0, Lon: 20.0}}

	nonHazmat, _, found := finder.NearestOnRoute(45.0, 20.0, "", nil, DefaultPreferredRadiusM, routePoints, 5000, false)
	if !found || nonHazmat.ID != parking.ID {
		t.Errorf("non-hazmat: expected plain nearest (parking, id=%d), got %+v", parking.ID, nonHazmat)
	}

	hazmatChoice, _, found := finder.NearestOnRoute(45.0, 20.0, "", nil, DefaultPreferredRadiusM, routePoints, 5000, true)
	if !found || hazmatChoice.ID != fuel.ID {
		t.Errorf("hazmat: expected the fuel station (id=%d) preferred over closer parking, got %+v", fuel.ID, hazmatChoice)
	}
}
