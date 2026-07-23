package algorithm

// AStar finds the minimum-cost path from start to goal using a straight-line
// (haversine) distance-to-goal heuristic. Admissible because no real road path
// can be shorter than the straight line, so A* here is guaranteed to return the
// same optimal cost as Dijkstra - just visiting fewer nodes on the way (verified
// in algorithm_test.go).
func AStar(g *Graph, start, goal int64, profile VehicleProfile) (Result, error) {
	goalNode, ok := g.Nodes[goal]
	if !ok {
		return Result{}, errNoPath
	}

	heuristic := func(node int64) float64 {
		n, ok := g.Nodes[node]
		if !ok {
			return 0
		}
		return haversineMeters(n.Lat, n.Lon, goalNode.Lat, goalNode.Lon)
	}

	return search(g, start, goal, profile, heuristic)
}
