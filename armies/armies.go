package armies

import (
	"errors"
	"fmt"

	"github.com/pgruenbacher/got/families"
	"github.com/pgruenbacher/got/regions"
)

type armyId string

type Armies map[armyId]*Army

type SupplyStatus string

const (
	CUTOFF_STARVING  SupplyStatus = "CUTOFF_STARVING"
	CUTOFF_SURVIVING SupplyStatus = "CUTOFF_SURVIVING"
)

type Army struct {
	Id             armyId
	Morale         int
	Strength       int
	Size           int
	SupplyStatus   SupplyStatus
	StartingRegion regions.RegionId
	HomeRegion     regions.RegionId
	Region         *regions.Region
	Home           *regions.Region
	House          families.HouseId
}

// Using config values to initialize the rest of the object
func (self Armies) Init(r regions.Regions) error {
	for _, army := range self {
		// declare starting regions
		if region, ok := r[army.StartingRegion]; ok {
			army.Region = region
		} else {
			return errors.New(fmt.Sprintf("army %v starting region not exist", army.Id))
		}
		// declare home region
		if region, ok := r[army.HomeRegion]; ok {
			army.Home = region
		} else {
			return errors.New(fmt.Sprintf("army %v home region not exist", army.Id))
		}
	}
	return nil
}

func (self Armies) EvalSupplies(r regions.Regions) error {
	for _, army := range self {
		supplied := army.EvalSupplyRoute(r)
		if !supplied {
			army.SupplyStatus = CUTOFF_STARVING
		}
	}
	return nil
}

func (self *Army) EvalSupplyRoute(r regions.Regions) bool {
	if self.Region == self.Home {
		return true
	}
	// shortPath := r.Path(self.Region.Id, self.Home.Id, nil, nil)
	// longPath := r.Path(self.Region.Id, self.Home.Id, hostileFilter, self)
	// if len(longPath) != len(shortPath) {
	// 	return false
	// }
	return true
}

func (self *Army) March(to *regions.Edge) error {
	if _, err := self.ValidateMarch(to); err != nil {
		return err
	}
	self.Region = to.Dst
	return nil
}

func (self *Army) ValidateMarch(edge *regions.Edge) (bool, error) {
	if self.Region.Id != edge.Src.Id {
		return false, errors.New(fmt.Sprintf("march invalid: army %v not located at %v", self.Id, edge.Src))
	}
	return true, nil
}

func newArmy(a *Army) *Army {
	b := *a
	return &b
}

var SampleArmies = `
[army1]
startingRegion="region3cost"
homeRegion="region1"
house="house1"

[army2]
startingRegion="region1"
homeRegion="region1"
house="house2"
`
