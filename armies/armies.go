package armies

import (
	"errors"
	"fmt"

	"gopkg.in/validator.v2"

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
	Morale         int `validate:"min=1,max=5"`
	Size           int `validate:"min=1,max=100"`
	Quality        int `validate:"min=1,max=5"`
	combatState    combatStatus
	SupplyState    SupplyStatus
	DefenseState   defenseStatus
	StartingRegion regions.RegionId `validate:"nonzero"`
	HomeRegion     regions.RegionId //May get rid of, want to use house reference
	Region         *regions.Region
	Home           *regions.Region
	House          families.HouseId `validate:"nonzero"`
}

func (self Army) Strength() int {
	return self.Morale + self.Size + self.Quality
}

// Using config values to initialize the rest of the object
func (self Armies) Init(r regions.Regions) error {
	for armyId, army := range self {
		// perform struct field validations
		if err := validator.Validate(army); err != nil {
			fmt.Println("error!")
			// values not valid, deal with errors here
			return err
		}
		// set army id using existing key
		army.Id = armyId
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
			army.SupplyState = CUTOFF_STARVING
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
morale = 3
size = 30
quality = 3

[army2]
startingRegion="region1"
homeRegion="region1"
house="house2"
morale = 4
size = 30
quality = 3
`
