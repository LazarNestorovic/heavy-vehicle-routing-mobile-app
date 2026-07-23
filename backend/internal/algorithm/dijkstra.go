package algorithm

import "container/heap"

type Result struct {
	Path []int64
	Cost float64
}

type pqItem struct {
	node     int64
	priority float64 // cost so far (Dijkstra) or cost so far + heuristic (A*)
}

type priorityQueue []pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].priority < pq[j].priority }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x any)        { *pq = append(*pq, x.(pqItem)) }
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

// Dijkstra finds the minimum-cost path from start to goal, only traversing edges
// `allowed` for the given vehicle profile.
func Dijkstra(g *Graph, start, goal int64, profile VehicleProfile) (Result, error) {
	return search(g, start, goal, profile, func(int64) float64 { return 0 })
}

// search is shared by Dijkstra (zero heuristic) and A* (haversine-to-goal heuristic).
func search(g *Graph, start, goal int64, profile VehicleProfile, heuristic func(node int64) float64) (Result, error) {
	dist := map[int64]float64{start: 0}
	prev := map[int64]int64{}

	pq := &priorityQueue{{node: start, priority: heuristic(start)}}
	heap.Init(pq)
	visited := map[int64]bool{}

	for pq.Len() > 0 {
		current := heap.Pop(pq).(pqItem).node
		if visited[current] {
			continue
		}
		visited[current] = true

		if current == goal {
			return Result{Path: reconstructPath(prev, start, goal), Cost: dist[goal]}, nil
		}

		for _, edge := range g.AdjList[current] {
			if !allowed(edge, profile) || visited[edge.To] {
				continue
			}
			newDist := dist[current] + cost(edge)
			if old, ok := dist[edge.To]; !ok || newDist < old {
				dist[edge.To] = newDist
				prev[edge.To] = current
				heap.Push(pq, pqItem{node: edge.To, priority: newDist + heuristic(edge.To)})
			}
		}
	}

	return Result{}, errNoPath
}

func reconstructPath(prev map[int64]int64, start, goal int64) []int64 {
	path := []int64{goal}
	for path[len(path)-1] != start {
		path = append(path, prev[path[len(path)-1]])
	}
	// reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
