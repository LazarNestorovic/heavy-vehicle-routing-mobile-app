package algorithm

import (
	"math"
	"testing"
)

// buildDiamondGraph creates a small deterministic graph:
//
//	1 --10m-- 2 --10m-- 4   (top path, unrestricted)
//	1 --5m--- 3 --5m---- 4   (bottom path, shorter, but height-restricted to 3.5m)
//
// so the "obviously shortest" path (via node 3) is only reachable by a low vehicle.
//
// All nodes share the same coordinates on purpose: real lat/lon deltas are on the
// order of kilometers, while these hand-picked edge costs are meters, so any
// non-trivial coordinate spread would make the haversine heuristic overestimate
// true remaining cost and break A*'s admissibility for this toy graph (that
// exact mistake is what the first version of this test caught). The real-data
// test below exercises A* with genuine, consistent coordinates and costs.
func buildDiamondGraph() *Graph {
	g := NewGraph()
	g.Nodes[1] = Node{ID: 1, Lat: 0, Lon: 0}
	g.Nodes[2] = Node{ID: 2, Lat: 0, Lon: 0}
	g.Nodes[3] = Node{ID: 3, Lat: 0, Lon: 0}
	g.Nodes[4] = Node{ID: 4, Lat: 0, Lon: 0}

	g.addEdge(1, Edge{To: 2, LengthM: 10})
	g.addEdge(2, Edge{To: 4, LengthM: 10})
	g.addEdge(2, Edge{To: 1, LengthM: 10})
	g.addEdge(4, Edge{To: 2, LengthM: 10})

	g.addEdge(1, Edge{To: 3, LengthM: 5, MaxHeightM: 3.5})
	g.addEdge(3, Edge{To: 4, LengthM: 5, MaxHeightM: 3.5})
	g.addEdge(3, Edge{To: 1, LengthM: 5, MaxHeightM: 3.5})
	g.addEdge(4, Edge{To: 3, LengthM: 5, MaxHeightM: 3.5})

	return g
}

func TestDijkstra_TakesShortestUnrestrictedPath(t *testing.T) {
	g := buildDiamondGraph()

	// A low vehicle (2.5m) can use the height-restricted shortcut via node 3.
	result, err := Dijkstra(g, 1, 4, VehicleProfile{HeightM: 2.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Cost != 10 {
		t.Errorf("expected cost 10 (via node 3), got %v (path %v)", result.Cost, result.Path)
	}
	if len(result.Path) != 3 || result.Path[1] != 3 {
		t.Errorf("expected path [1 3 4], got %v", result.Path)
	}
}

func TestDijkstra_ExcludesEdgeAboveHeightLimit(t *testing.T) {
	g := buildDiamondGraph()

	// A tall vehicle (4.0m) must detour via node 2, even though it's longer.
	result, err := Dijkstra(g, 1, 4, VehicleProfile{HeightM: 4.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Cost != 20 {
		t.Errorf("expected cost 20 (forced detour via node 2), got %v (path %v)", result.Cost, result.Path)
	}
	if len(result.Path) != 3 || result.Path[1] != 2 {
		t.Errorf("expected path [1 2 4], got %v", result.Path)
	}
}

func TestDijkstra_NoPathWhenAllEdgesExcluded(t *testing.T) {
	g := NewGraph()
	g.Nodes[1] = Node{ID: 1}
	g.Nodes[2] = Node{ID: 2}
	g.addEdge(1, Edge{To: 2, LengthM: 5, MaxHeightM: 3.0})

	_, err := Dijkstra(g, 1, 2, VehicleProfile{HeightM: 4.0})
	if err == nil {
		t.Fatal("expected errNoPath, got nil")
	}
}

// TestAStar_MatchesDijkstra checks the property that makes A* trustworthy here:
// with an admissible heuristic, it must find the same optimal cost as Dijkstra,
// on every profile that changes which edges are excluded.
func TestAStar_MatchesDijkstra(t *testing.T) {
	g := buildDiamondGraph()

	for _, profile := range []VehicleProfile{{HeightM: 2.5}, {HeightM: 4.0}} {
		dijkstraResult, err := Dijkstra(g, 1, 4, profile)
		if err != nil {
			t.Fatalf("dijkstra: unexpected error: %v", err)
		}
		astarResult, err := AStar(g, 1, 4, profile)
		if err != nil {
			t.Fatalf("astar: unexpected error: %v", err)
		}
		if astarResult.Cost != dijkstraResult.Cost {
			t.Errorf("profile %+v: A* cost %v != Dijkstra cost %v", profile, astarResult.Cost, dijkstraResult.Cost)
		}
	}
}

// buildNodeBarrierGraph is buildDiamondGraph's shape, but the shortcut's
// height restriction sits on the NODE (node 3 itself), not on its edges -
// exercises nodeAllowed rather than allowed (see documentations/guides/
// extract-osm-corridor.md for why this needed its own synthetic test: real
// height-tagged barrier nodes turned out to not be connected to any major
// road in the actual corridor extract).
func buildNodeBarrierGraph() *Graph {
	g := NewGraph()
	g.Nodes[1] = Node{ID: 1, Lat: 0, Lon: 0}
	g.Nodes[2] = Node{ID: 2, Lat: 0, Lon: 0}
	g.Nodes[3] = Node{ID: 3, Lat: 0, Lon: 0, MaxHeightM: 3.5}
	g.Nodes[4] = Node{ID: 4, Lat: 0, Lon: 0}

	g.addEdge(1, Edge{To: 2, LengthM: 10})
	g.addEdge(2, Edge{To: 4, LengthM: 10})
	g.addEdge(2, Edge{To: 1, LengthM: 10})
	g.addEdge(4, Edge{To: 2, LengthM: 10})

	g.addEdge(1, Edge{To: 3, LengthM: 5})
	g.addEdge(3, Edge{To: 4, LengthM: 5})
	g.addEdge(3, Edge{To: 1, LengthM: 5})
	g.addEdge(4, Edge{To: 3, LengthM: 5})

	return g
}

func TestDijkstra_NodeBarrierAllowsLowVehicleShortcut(t *testing.T) {
	g := buildNodeBarrierGraph()

	result, err := Dijkstra(g, 1, 4, VehicleProfile{HeightM: 2.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Cost != 10 {
		t.Errorf("expected cost 10 (via node 3), got %v (path %v)", result.Cost, result.Path)
	}
	if len(result.Path) != 3 || result.Path[1] != 3 {
		t.Errorf("expected path [1 3 4], got %v", result.Path)
	}
}

func TestDijkstra_NodeBarrierExcludesTallVehicle(t *testing.T) {
	g := buildNodeBarrierGraph()

	// Node 3's edges (1->3, 3->4) carry no height tag of their own - only the
	// node itself does. A tall vehicle must still be forced to detour via
	// node 2, proving the exclusion is coming from nodeAllowed, not allowed.
	result, err := Dijkstra(g, 1, 4, VehicleProfile{HeightM: 4.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Cost != 20 {
		t.Errorf("expected cost 20 (forced detour via node 2), got %v (path %v)", result.Cost, result.Path)
	}
	if len(result.Path) != 3 || result.Path[1] != 2 {
		t.Errorf("expected path [1 2 4], got %v", result.Path)
	}
}

func TestDijkstra_PrefersHigherRoadClassWhenLengthEqual(t *testing.T) {
	g := NewGraph()
	g.Nodes[1] = Node{ID: 1, Lat: 0, Lon: 0}
	g.Nodes[2] = Node{ID: 2, Lat: 0, Lon: 0}
	g.Nodes[3] = Node{ID: 3, Lat: 0, Lon: 0}
	g.Nodes[4] = Node{ID: 4, Lat: 0, Lon: 0}

	g.addEdge(1, Edge{To: 2, LengthM: 10, RoadClass: "motorway"})
	g.addEdge(2, Edge{To: 4, LengthM: 10, RoadClass: "motorway"})

	g.addEdge(1, Edge{To: 3, LengthM: 10, RoadClass: "secondary"})
	g.addEdge(3, Edge{To: 4, LengthM: 10, RoadClass: "secondary"})

	result, err := Dijkstra(g, 1, 4, VehicleProfile{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Path) != 3 || result.Path[1] != 2 {
		t.Errorf("expected motorway path via node 2 despite equal raw length, got %v", result.Path)
	}
	if want := 20 * roadClassMultiplier["motorway"]; result.Cost != want {
		t.Errorf("expected cost %v (motorway multiplier applied), got %v", want, result.Cost)
	}
}

// TestDijkstra_TurnPenaltyPrefersStraightOverShorterSharpTurn uses real,
// distinct coordinates (unlike the diamond graph's shared Lat:0,Lon:0 nodes,
// which make every turn angle 0) to exercise turnPenaltyMeters. Path A runs
// straight through a collinear midpoint (turn angle 0, raw length 19). Path B
// is a shorter raw distance (17) but turns ~97 degrees at its midpoint - a
// "sharp turn" per turnPenaltyMeters's bucket, which should push its total
// cost above path A's.
func TestDijkstra_TurnPenaltyPrefersStraightOverShorterSharpTurn(t *testing.T) {
	g := NewGraph()
	g.Nodes[1] = Node{ID: 1, Lat: 0, Lon: 0}
	g.Nodes[4] = Node{ID: 4, Lat: 0, Lon: 2}
	g.Nodes[10] = Node{ID: 10, Lat: 0, Lon: 1} // collinear midpoint - straight path
	g.Nodes[20] = Node{ID: 20, Lat: 1, Lon: 0.5}

	g.addEdge(1, Edge{To: 10, LengthM: 9})
	g.addEdge(10, Edge{To: 4, LengthM: 10})

	g.addEdge(1, Edge{To: 20, LengthM: 8})
	g.addEdge(20, Edge{To: 4, LengthM: 9})

	result, err := Dijkstra(g, 1, 4, VehicleProfile{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Path) != 3 || result.Path[1] != 10 {
		t.Errorf("expected the straight (turn-free) path via node 10 despite its longer raw length, got %v", result.Path)
	}
	if result.Cost != 19 {
		t.Errorf("expected cost 19 (no turn penalty on the collinear path), got %v", result.Cost)
	}
}

func TestParseMeters(t *testing.T) {
	cases := map[string]float64{
		"4.5":     4.5,
		"4.5 m":   4.5,
		"default": 0,
		"":        0,
		"garbage": 0,
	}
	for input, want := range cases {
		if got := parseMeters(input); got != want {
			t.Errorf("parseMeters(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestHaversineMeters_KnownDistance(t *testing.T) {
	// Belgrade to Novi Sad city centers, straight line, ~70km.
	d := haversineMeters(44.7866, 20.4489, 45.2671, 19.8335)
	if d < 65000 || d > 80000 {
		t.Errorf("expected ~70km straight-line distance, got %.1fkm", d/1000)
	}
}

func TestHaversineMeters_ZeroForSamePoint(t *testing.T) {
	if d := haversineMeters(44.8, 20.4, 44.8, 20.4); math.Abs(d) > 1e-9 {
		t.Errorf("expected 0 for identical points, got %v", d)
	}
}

// --- Real-data tests: load the actual Belgrade-Novi Sad corridor extract (major
// roads only - see documentations/guides/extract-osm-corridor.md) and run the
// algorithm against it, the same coordinates used throughout the Valhalla work
// in documentations/features/2026-07-21-risk-scoring-layer.md.

const corridorFixture = "testdata/beograd-novisad-corridor.osm"

func loadCorridor(t *testing.T) *Graph {
	t.Helper()
	g, err := LoadOSMXML(corridorFixture)
	if err != nil {
		t.Fatalf("load %s: %v (run the extraction steps in documentations/guides/extract-osm-corridor.md)", corridorFixture, err)
	}
	if len(g.Nodes) < 1000 {
		t.Fatalf("suspiciously small graph: %d nodes - is %s the right extract?", len(g.Nodes), corridorFixture)
	}
	return g
}

func TestRealCorridor_DijkstraFindsPlausibleRoute(t *testing.T) {
	g := loadCorridor(t)

	start, err := g.NearestNode(44.8, 20.4) // Belgrade
	if err != nil {
		t.Fatalf("nearest node to origin: %v", err)
	}
	goal, err := g.NearestNode(45.25, 19.85) // Novi Sad
	if err != nil {
		t.Fatalf("nearest node to destination: %v", err)
	}

	result, err := Dijkstra(g, start, goal, VehicleProfile{HeightM: 4.0, WeightKg: 40000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distanceKm := result.Cost / 1000
	// Valhalla found 84.8-92.7km for this pair (see risk-scoring-layer.md); this
	// module only has major roads and a straight-line-to-nearest-node start/end,
	// so a wider but still sane band.
	if distanceKm < 60 || distanceKm > 150 {
		t.Errorf("expected a plausible Belgrade-Novi Sad distance, got %.1fkm (path length %d)", distanceKm, len(result.Path))
	}
	t.Logf("Dijkstra: %.1fkm across %d nodes", distanceKm, len(result.Path))
}

// TestRealCorridor_HeightRestrictionExcludesRealTaggedEdge uses a real maxheight=4.3
// way found in the corridor extract (near central Belgrade, ~44.808,20.429) to prove
// the exclusion mechanism on genuine OSM data, not just the synthetic diamond graph.
//
// Note this is a *different* case from the Novi Banovci detour investigated live in
// documentations/features/2026-07-21-risk-scoring-layer.md: this bounded, major-roads-only
// extract does not contain a maxheight tag near Novi Banovci at all, so it can't
// reproduce that exact scenario. Most likely explanations (documented in
// documentations/features/2026-07-21-bounded-astar-dijkstra.md): that restriction may be
// a node-level barrier tag rather than a way-level maxheight tag (this loader only
// reads way tags), and/or Valhalla's real route through there isn't the geometrically
// shortest path our simple distance-based cost function picks, so the tagged edge
// there just isn't on our chosen path. This test sidesteps that by picking two points
// that are only connected, in this reduced graph, through one specific tagged edge.
func TestRealCorridor_HeightRestrictionExcludesRealTaggedEdge(t *testing.T) {
	g := loadCorridor(t)

	start, err := g.NearestNode(44.8110, 20.4290)
	if err != nil {
		t.Fatalf("nearest node to start: %v", err)
	}
	goal, err := g.NearestNode(44.8060, 20.4290)
	if err != nil {
		t.Fatalf("nearest node to goal: %v", err)
	}

	fits, err := Dijkstra(g, start, goal, VehicleProfile{HeightM: 4.0})
	if err != nil {
		t.Fatalf("4.0m vehicle: expected a path under the 4.3m limit, got error: %v", err)
	}
	t.Logf("height=4.0m -> %.0fm", fits.Cost)

	_, err = Dijkstra(g, start, goal, VehicleProfile{HeightM: 4.5})
	if err == nil {
		t.Error("4.5m vehicle: expected no path (exceeds the real 4.3m maxheight tag on this edge, and this bounded major-roads-only extract has no alternate), but a route was found")
	}
}

// TestRealCorridor_ParsesNodeBarrierHeightTag verifies node-tag parsing
// against genuine OSM data (not the synthetic graph used by
// TestDijkstra_NodeBarrierExcludesTallVehicle). Node 11742525355 is a real
// barrier=lift_gate, maxheight=2.2 node near central Belgrade
// (44.8056573,20.449473) - see documentations/guides/extract-osm-corridor.md
// for why this node (and the only two others like it in this extract) isn't
// actually connected to any of the major-road edges this module routes over,
// so this test covers parsing correctness only, not exclusion behavior.
func TestRealCorridor_ParsesNodeBarrierHeightTag(t *testing.T) {
	g := loadCorridor(t)

	n, ok := g.Nodes[11742525355]
	if !ok {
		t.Fatal("expected node 11742525355 to be present in the loaded graph")
	}
	if n.MaxHeightM != 2.2 {
		t.Errorf("expected MaxHeightM 2.2 parsed from the node's own maxheight tag, got %v", n.MaxHeightM)
	}
}

func TestRealCorridor_AStarMatchesDijkstra(t *testing.T) {
	g := loadCorridor(t)

	start, _ := g.NearestNode(44.8, 20.4)
	goal, _ := g.NearestNode(45.25, 19.85)

	dijkstraResult, err := Dijkstra(g, start, goal, VehicleProfile{HeightM: 4.0})
	if err != nil {
		t.Fatalf("dijkstra: %v", err)
	}
	astarResult, err := AStar(g, start, goal, VehicleProfile{HeightM: 4.0})
	if err != nil {
		t.Fatalf("astar: %v", err)
	}

	if math.Abs(astarResult.Cost-dijkstraResult.Cost) > 1 { // 1m tolerance for float accumulation
		t.Errorf("A* cost %.1f != Dijkstra cost %.1f on real data", astarResult.Cost, dijkstraResult.Cost)
	}
}
