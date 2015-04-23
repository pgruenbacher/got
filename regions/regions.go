package regions

import (
	"errors"
	"fmt"
)

type Regions map[RegionId]*Region

type RegionId string

var (
	NeighborItself    = errors.New("region can't neighbor itself")
	NoNeighbors       = errors.New("region has no neighbors")
	NeighborsMismatch = errors.New("region neighbor doesn't share neighbor id")
	NeighborNonexist  = errors.New("region neighbor id doesn't exist")
)

/*
Nodes contain units, and are connected by edges.
*/
type Region struct {
	// Id is the id of the region
	Id RegionId
	// Size is the number of units supported by the region.
	// Overcapacity will lead to penalties, especially if supply route is cutof
	Capacity int
	// Edges go from this region to others.
	Edges map[RegionId]*Edge
	// terrain type
	Terrain Terrain
	// Capfacity
	Neighbors []RegionId
	// Rivers    []RegionId
	// Walls     []RegionId
}

type Terrain string

const (
	Mountain Terrain = "mountain"
	Plain    Terrain = "plain"
)

/*
Edge goes from one node to another.
*/
type Edge struct {
	// Src is the id of the source node.
	Src *Region
	// Dst is the id of the destination node.
	Dst *Region
	// Whether there is a river/mountain/wall between the regions.
	Boundary Boundary
}

func (self Regions) initializeAll() bool {
	for regionId, region := range self {
		region.Edges = make(map[RegionId]*Edge, len(region.Neighbors))
		region.Id = regionId
	}
	return true
}

func (self Regions) ConnectAll() error {
	if ok := self.initializeAll(); !ok {
		return errors.New("couldn't initialize")
	}
	for regionId, region := range self {
		if regionId != region.Id {
			return NeighborsMismatch
		}
		if len(region.Neighbors) == 0 {
			return NoNeighbors
		}
		for _, neighborId := range region.Neighbors {
			if neighbor, ok := self[neighborId]; ok {
				if neighbor.Id == region.Id {
					return NeighborItself
				}
				err := validateNeighbor(region, neighbor)
				if err != nil {
					return err
				}
				err = region.Connect(neighbor)
				if err != nil {
					return err
				}
			} else {
				return NeighborNonexist
			}
		}
	}
	return nil
}

func (self *Region) Connect(region *Region) error {
	away := &Edge{
		Src: self,
		Dst: region,
	}
	here := &Edge{
		Src: region,
		Dst: self,
	}
	self.Edges[region.Id] = away
	region.Edges[self.Id] = here
	return nil
}

func validateNeighbor(a, b *Region) error {
	if valid := checkNeighbors(a.Id, b.Neighbors); !valid {
		return errors.New(fmt.Sprintf("neighbor %v doesn't reference region %v", a.Id, b.Id))
	}
	return nil
}

func checkNeighbors(a RegionId, list []RegionId) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

var ExampleRegions string = `
    [region1]
    size = 3
    neighbors = ["region2","region4","region2cost"]


    [region2]
    size = 3
    terrain = "plain"
    neighbors = ["region1","region3"]

    [region2cost]
    terrain="mountain"
    neighbors=["region1","region3cost"]

    [region3cost]
    terrain="mountain"
    neighbors=["region2cost","region6"]


    [region3]
    size = 3
    neighbors = ["region2","region7"]

    [region7]
    size = 3 
    neighbors =["region3","region6"]

    [region4]
    size = 3
    terrain = "plain"
    neighbors = ["region1","region5"]


    [region5]
    size = 3
    neighbors = ["region4"]

    [region6]
    size = 3
    neighbors = ["region7","region3cost"]
    `

var ExampleRivers string = `
    [yellowfork]
    name="yellow fork"
    borders = ["region2","region1","region1","region4"]
    `
