package regions

import (
	"errors"
)

var (
	OddBorderNumber         = errors.New("invalid number of borders for boundary")
	BorderDoesNotExist      = errors.New("errors does not exist")
	InvalidBoundaryBorders  = errors.New("invalid boundary border pairing")
	BoundaryAlreadyAssigned = errors.New("boundary already assigned")
)

type Boundary interface {
	borders() []RegionId
}

type Boundaries interface {
	boundaries() []Boundary
}

type River struct {
	Name    string
	Borders []RegionId
}

func (r River) borders() []RegionId {
	return r.Borders
}

type Rivers map[string]*River

type Wall struct {
	Name    string
	Borders []RegionId
}

func (r Wall) borders() []RegionId {
	return r.Borders
}

type Walls map[string]*Wall

func (self Regions) IncorporateBoundary(b Boundary) error {
	if len(b.borders())%2 != 0 {
		return OddBorderNumber
	}
	for i := 0; i < len(b.borders()); i++ {
		if i%2 == 1 {
			continue
		}

		border1 := b.borders()[i]
		border2 := b.borders()[i+1]

		region1, ok := self[border1]
		if !ok {
			return BorderDoesNotExist
		}
		region2, ok := self[border2]
		if !ok {
			return BorderDoesNotExist
		}

		valid := false
		for _, edge1 := range region1.Edges {
			if edge1.Dst.Id == region2.Id {
				for _, edge2 := range edge1.Dst.Edges {
					if edge2.Dst.Id == region1.Id {
						if edge1.Boundary != nil || edge2.Boundary != nil {
							return BoundaryAlreadyAssigned
						}
						// need to assign this way
						// region1.Edges[edge1.Dst.Id].Boundary = b
						// region2.Edges[edge2.Dst.Id].Boundary = a
						edge1.Boundary = b
						edge2.Boundary = b

						valid = true
					}
				}
			}
		}
		if !valid {
			return InvalidBoundaryBorders
		}

	}
	return nil
}