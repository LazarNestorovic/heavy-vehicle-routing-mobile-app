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
