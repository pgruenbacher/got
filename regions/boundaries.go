package regions

import (
	"errors"
	"fmt"
)

var (
	OddBorderNumber         = errors.New("invalid number of borders for boundary")
	BorderDoesNotExist      = errors.New("errors does not exist")
	InvalidBoundaryBorders  = errors.New("invalid boundary border pairing")
	BoundaryAlreadyAssigned = errors.New("boundary already assigned")
)

type Boundary interface {
	borders() []RegionId
	MovePenalty() int
	AttackPenalty() float32
}

type Boundaries interface {
	boundaries() []Boundary
}

type River struct {
	Name             string
	Borders          []RegionId
	MovementPenalty  int     `toml:"movement_penalty"`
	AttackingPenalty float32 `toml:"attack_penalty"`
}

func (r River) String() string {
	return r.Name
}

func (r River) borders() []RegionId {
	return r.Borders
}

func (r River) MovePenalty() int {
	return r.MovementPenalty
}

func (self River) AttackPenalty() float32 {
	return self.AttackingPenalty
}

type Rivers map[string]*River

type Wall struct {
	Name             string
	Borders          []RegionId
	MovementPenalty  int     `toml:"movement_penalty"`
	AttackingPenalty float32 `toml:"attack_penalty"`
}

func (r Wall) borders() []RegionId {
	return r.Borders
}

func (r Wall) MovePenalty() int {
	return r.MovementPenalty
}

func (self Wall) AttackPenalty() float32 {
	return self.AttackingPenalty
}

type Walls map[string]*Wall

type NoBoundary bool

func (NoBoundary) MovePenalty() int {
	return 0
}

func (NoBoundary) AttackPenalty() float32 {
	return 0
}
func (NoBoundary) borders() []RegionId {
	return nil
}

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
						switch edge1.Boundary.(type) {
						case *NoBoundary:
							switch edge2.Boundary.(type) {
							case *NoBoundary:
								edge1.Boundary = b
								edge2.Boundary = b
								valid = true
							default:
								return errors.New(fmt.Sprintf("edge %v to %v already has boundary assigned", edge2.Src.Id, edge2.Dst.Id))
							}
						default:
							return errors.New(fmt.Sprintf("edge %v to %v already has boundary assigned", edge1.Src.Id, edge1.Dst.Id))
						}
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

var ExampleRivers string = `
    [yellowfork]
    name="yellow fork"
    borders = ["region2","region1","region1","region4"]
    movement_penalty = 2
    attacing_penalty = -0.2
    `
