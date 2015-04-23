package armies

import (
	"errors"
	"fmt"

	"github.com/pgruenbacher/got/actions"
	"github.com/pgruenbacher/got/diplomats"
	"github.com/pgruenbacher/got/events"
	"github.com/pgruenbacher/got/regions"
)

// Orders
// Generic Army order has army id
type ArmyOrder struct {
	actions.Order
	ArmyId armyId
}

// specific orders
type MarchOrder struct {
	ArmyOrder
	Src regions.RegionId
	Dst regions.RegionId
	// context of order
	Ctx Context
}

type Context string

const (
	// Order and Event Contexts
	RETREAT         Context = "RETREAT"
	ATTACK          Context = "ATTACK"
	REDIRECT_ATTACK Context = "REDIRECT_ATTACK"
	MARCH           Context = "MARCH"
	/*
	 *  Event Contexts
	 */
	// retreat as enemy attacks
	STRATEGIC_RETREAT Context = "STRATEGIC_RETREAT"
	// attack, but all enemy marched elsewhere
	ATTACK_PURSUIT Context = "ATTACK_PURSUIT"
	// unexpected attack,
)

// Events
// Generic army event has army id
type ArmyEvent struct {
	events.Event
	ArmyId armyId
}

// specific events
type MarchEvent struct {
	ArmyEvent
	Src regions.RegionId
	Dst regions.RegionId
}

type AttackEvent struct {
	ArmyEvent
	Src regions.RegionId
	Dst regions.RegionId
}

// manager structeure
type ArmiesManager struct {
	Armies    Armies
	regions   regions.Regions
	diplomacy diplomats.DiplomatsTable
	Config    Config
}

type Config struct {
	Penalties []Penalty
}

type Penalty struct {
	Boundary        regions.Boundary
	CrossingPenalty int
}

/*
 * End Type Section
 *
 */

// Armies Manager methods
func (self *ArmiesManager) Init(a Armies, r regions.Regions, d diplomats.DiplomatsTable) error {
	self.regions = r
	self.diplomacy = d
	self.Armies = a
	self.Armies.Init(r)
	return nil
}

func (self *ArmiesManager) EvaluateArmies() error {
	if err := self.Armies.EvalSupplies(self.regions); err != nil {
		return err
	}
	return nil

}

func (self *ArmiesManager) ReadOrders(orders interface{}) (events events.EventsInterface, err error) {
	err = nil

	switch t := orders.(type) {
	default:
		// do nothing
	case []MarchOrder:
		events, err = self.marchOrders(t)
	}
	return events, err
}

func (self *ArmiesManager) GivePossibleOrders(id armyId) (orders []actions.OrderInterface) {
	army := self.Armies[id]
outerloop:
	for _, to := range army.Region.Edges {
		for _, otherArmy := range self.armiesWithin(to.Dst) {
			if self.diplomacy.IsEnemy(army.House, otherArmy.House) {
				attackOrder := MarchOrder{
					ArmyOrder: newArmyOrder(army.Id),
					Src:       to.Src.Id,
					Dst:       to.Dst.Id,
					Ctx:       ATTACK,
				}
				orders = append(orders, attackOrder)
				// skip the move order append
				continue outerloop
			}
		}
		moveOrder := MarchOrder{
			ArmyOrder: newArmyOrder(army.Id),
			Src:       to.Src.Id,
			Dst:       to.Dst.Id,
			Ctx:       MARCH,
		}
		orders = append(orders, moveOrder)
	}
	return orders
}

/*
 * individual order handling
 *
 */

func (self *ArmiesManager) marchOrders(orders []MarchOrder) (e []MarchEvent, err error) {
	if err = self.validateMarchOrders(orders); err != nil {
		return e, err
	}
	for _, order := range orders {
		// get the army to perform on
		army := self.Armies[order.ArmyId]
		// Perform the march
		edge := army.Region.Edges[order.Dst]
		if err = army.March(edge); err != nil {
			return e, err
		}
		// make the event
		event := MarchEvent{
			ArmyEvent: newArmyEvent(army.Id),
			Src:       edge.Src.Id,
			Dst:       edge.Dst.Id,
		}
		e = append(e, event)

	}
	return e, nil
}

/*
 * Miscellaneous
 *
 */
func (self *ArmiesManager) armiesWithin(region *regions.Region) (armies []*Army) {
	for _, army := range self.Armies {
		if army.Region == region {
			armies = append(armies, army)
		}
	}
	return armies
}

func (self *ArmiesManager) checkDestinations(orders []MarchOrder) {
	dstMap := make(map[regions.RegionId]armyId)
	for _, order := range orders {
		if _, ok := dstMap[order.Dst]; !ok {
			dstMap[order.Dst] = order.ArmyId
			// skip to next order
			continue
		}
		// there is an army that is also moving here, find out if it is hostile
		otherArmyId := dstMap[order.Dst]
		if self.diplomacy.IsEnemy(self.Armies[order.ArmyId].House, self.Armies[otherArmyId].House) {

		}
		if self.diplomacy.IsEnemy(self.Armies[order.ArmyId].House, self.Armies[otherArmyId].House) {

		}
	}
}

/*
 * Validation Section
 *
 */

func (self *ArmiesManager) validateMarchOrders(orders []MarchOrder) error {
	for _, order := range orders {
		// validate the army id
		if err := self.validateArmyOrder(order.ArmyOrder); err != nil {
			return err
		}
		army := self.Armies[order.ArmyId]
		// validate src region
		if err := self.validateSrc(army, order); err != nil {
			return err
		}
		// validate destination region
		if err := self.validateDestination(army, order); err != nil {
			return err
		}

	}
	return nil
}

func (self *ArmiesManager) validateArmyOrder(order ArmyOrder) error {
	if _, ok := self.Armies[order.ArmyId]; !ok {
		return errors.New(fmt.Sprintf("order %v had invalid armyId %v", order.Id, order.ArmyId))
	}
	return nil
}

func (self *ArmiesManager) validateSrc(army *Army, order MarchOrder) error {
	if _, ok := self.regions[order.Src]; !ok {
		return errors.New(fmt.Sprintf("invalid src id %v", order.Src))
	}
	if army.Region.Id != order.Src {
		return errors.New(fmt.Sprintf("army region %v doesn't match src %v", army.Region.Id, order.Src))
	}
	return nil
}

func (self *ArmiesManager) validateDestination(army *Army, order MarchOrder) error {
	valid := false
	_, ok := self.regions[order.Dst]
	if !ok {
		return errors.New(fmt.Sprintf("invalid destination id %v", order.Dst))
	}
	for _, edge := range army.Region.Edges {
		if edge.Dst.Id == order.Dst {
			valid = true
		}
	}
	if !valid {
		return errors.New(fmt.Sprintf("none of the army region  %v edges match army destination %v", army.Region.Id, order.Dst))
	}

	return nil
}

/*
 *
 * Utilities
 *
 */

func newArmyEvent(id armyId) ArmyEvent {
	return ArmyEvent{
		Event:  events.NewEvent(),
		ArmyId: id,
	}
}

func newArmyOrder(id armyId) ArmyOrder {
	return ArmyOrder{
		Order:  actions.NewOrder(),
		ArmyId: id,
	}
}

var SampleArmies = `
[army1]
startingRegion="region3"
homeRegion="region1"
house="house1"

[army2]
startingRegion="region2"
homeRegion="region1"
house="house2"
`

var SampleMarchOrder = MarchOrder{
	ArmyOrder: newArmyOrder("army1"),
	Src:       "region3",
	Dst:       "region2",
}

var SampleAttackOrder = MarchOrder{
	ArmyOrder: newArmyOrder("army1"),
	Src:       "region1",
	Dst:       "region2",
}
