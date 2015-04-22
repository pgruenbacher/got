package regions

import (
	"container/heap"
	"fmt"
)

type PathFilter func(*Region) bool

type WeightFilter func(a, b *Region) int

type pathStep struct {
	path []RegionId
	pos  RegionId
}

/*
Path will return the shortest path between src and dst in self, discounting all paths that don't match the filter. A nil filter matches all nodes.
*/
func (self Regions) Path(src, dst RegionId, filter PathFilter) (result []RegionId) {
	// queue of paths to try
	queue := []pathStep{
		pathStep{
			path: nil,
			pos:  src,
		},
	}
	// found shortest paths to the regions
	paths := map[RegionId][]RegionId{
		src: nil,
	}
	// next step preallocated
	step := pathStep{}
	// best path to the dest so far
	var best []RegionId
	// as long as we have new paths to try
	for len(queue) > 0 {
		// pick first path to try
		step = queue[0]
		// pop the queue
		queue = queue[1:]
		// if the region actually exists
		if region, found := self[step.pos]; found {
			// for each edge from the region
			for _, edge := range region.Edges {
				// if we either haven't been where this edge leads before, or we would get there along a shorter path this time (*1)
				if lastPathHere, found := paths[edge.Dst.Id]; !found || len(step.path)+1 < len(lastPathHere) {
					// if we either haven't found dst yet, or if following this path is shorter than where we found dst
					if best == nil || len(step.path)+1 < len(best) {
						// if we aren't filtering region, or this region matches the filter
						if filter == nil || filter(region) {
							// make a new path that is the path here + this region + the edge we want to follow
							thisPath := make([]RegionId, len(step.path)+1)
							// copy the path to here to the new path
							copy(thisPath, step.path)
							// add this region
							thisPath[len(step.path)] = edge.Dst.Id
							// remember that this is the best way so far (guaranteed by *1)
							paths[edge.Dst.Id] = thisPath
							// if this path leads to dst
							if edge.Dst.Id == dst {
								best = thisPath
							}
							// queue up following this path further
							queue = append(queue, pathStep{
								path: thisPath,
								pos:  edge.Dst.Id,
							})
						}
					}
				}
			}
		}
	}
	return paths[dst]
}

func (self Regions) Djikstra(src, dst RegionId, weigher WeightFilter) (map[RegionId]RegionId, map[RegionId]int) {
	// queue of paths to try
	frontier := new(PriorityQueue)
	frontier.put(src, 0)
	// take a guess at the size
	cameFrom := make(map[RegionId]RegionId, 100)
	costSoFar := make(map[RegionId]int, 100)

	iter := 0
	for frontier.Len() > 0 {
		current := frontier.Pop().(*Item).region
		if current == dst {
			break
		}
		for _, edge := range self[current].Edges {
			newCost := costSoFar[current]
			newCost = newCost + weigher(edge.Src, edge.Dst)
			if dstCost, ok := costSoFar[edge.Dst.Id]; ok {
				fmt.Println(newCost, dstCost)
				// if new cost is greater than the existing dst cost, don't overwrite it
				if newCost > dstCost {
					continue
				}
			}
			costSoFar[edge.Dst.Id] = newCost
			frontier.put(edge.Dst.Id, newCost)
			cameFrom[edge.Dst.Id] = current
		}

		iter++
		if iter > 150 {
			break
		}
	}
	return cameFrom, costSoFar
}

type Item struct {
	region   RegionId // The value of the item; arbitrary.
	priority int      // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) put(id RegionId, priority int) {
	pq.Push(&Item{
		region:   id,
		priority: priority,
	})
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, region RegionId, priority int) {
	item.region = region
	item.priority = priority
	heap.Fix(pq, item.index)
}
