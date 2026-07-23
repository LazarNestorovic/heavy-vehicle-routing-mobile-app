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
