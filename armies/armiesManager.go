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
	TerrainPenalties TerrainPenalties
}

type TerrainPenalties map[regions.Terrain]TerrainPenalty
type TerrainPenalty map[regions.Terrain]Penalty
type Penalty struct {
	MovementPenalty int
	AttackPenalty   int
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

func (self ArmiesManager) marchPrioritize(order MarchOrder) int {
	var terrainPenalty, boundaryPenalty int
	edge := self.regions[order.Src].Edges[order.Dst]
	if f, ok := self.Config.TerrainPenalties[edge.Src.Terrain]; ok {
		if p, ok := f[edge.Dst.Terrain]; ok {
			terrainPenalty = p.MovementPenalty
		}
	}
	boundaryPenalty = edge.Boundary.Penalty()
	return terrainPenalty + boundaryPenalty
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
		for _, otherArmy := range armiesWithin(self.Armies, to.Dst) {
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
	self.checkDestinations(orders)
	// for _, order := range orders {
	// 	// get the army to perform on
	// 	army := self.Armies[order.ArmyId]
	// 	// Perform the march
	// 	edge := army.Region.Edges[order.Dst]
	// 	if err = army.March(edge); err != nil {
	// 		return e, err
	// 	}
	// 	// make the event
	// 	event := MarchEvent{
	// 		ArmyEvent: newArmyEvent(army.Id),
	// 		Src:       edge.Src.Id,
	// 		Dst:       edge.Dst.Id,
	// 	}
	// 	e = append(e, event)

	// }
	return e, nil
}

/*
 * Miscellaneous
 *
 */
func armiesWithin(a Armies, region *regions.Region) (armies []*Army) {
	for _, army := range a {
		if army.Region == region {
			armies = append(armies, army)
		}
	}
	return armies
}

func (self *ArmiesManager) checkDestinations(orders []MarchOrder) (e []MarchEvent, err error) {
	// create a temporary copy of the armies and their future destinations.
	tmpArmies := make(Armies, len(self.Armies))
	copyArmies(tmpArmies, self.Armies)
	// perform movement penalties and army prioritizations for moves, then return queue of armies
	pq := new(PriorityQueue)
	for idx, order := range orders {
		pq.Push(&Item{
			value:    idx,
			priority: 100 - self.marchPrioritize(order),
		})
	}

	// dstMap := make(map[regions.RegionId]armyId)
outerLoop:
	for pq.Len() > 0 {
		fmt.Println("as")
		order := orders[pq.Pop().(*Item).value]
		// innerLoop:
		for _, army := range armiesWithin(tmpArmies, self.regions[order.Dst]) {
			if self.diplomacy.IsEnemy(tmpArmies[order.ArmyId].House, army.House) {
				// initiate attack on house, cancel the other army's march order if it has one
				fmt.Println("attack!")
				continue outerLoop

			}
			if self.diplomacy.IsAlly(tmpArmies[order.ArmyId].House, army.House) {
				// army may move into region if there is an ally and there is no military
				fmt.Println("allies! move in!")
				continue outerLoop
			}
			// ELSE there is a neutral army present, cannot enter the region
			// cancel army order
			fmt.Println("cancel order")
			continue outerLoop
		}
		// if no armies present, then move in!
		army := tmpArmies[order.ArmyId]
		army.March(army.Region.Edges[order.Dst])
		fmt.Println("move army!")
	}
	return nil, nil
}

/*
 * Validation Section
 *
 */

func (self *ArmiesManager) validateMarchOrders(orders []MarchOrder) error {
outerloop:
	for _, order := range orders {
		// validate the army id
		if _, ok := self.Armies[order.ArmyId]; !ok {
			return errors.New(fmt.Sprintf("order %v had invalid armyId %v", order.Id, order.ArmyId))
		}
		army := self.Armies[order.ArmyId]
		// validate src region
		if _, ok := self.regions[order.Src]; !ok {
			return errors.New(fmt.Sprintf("invalid src id %v", order.Src))
		}
		if army.Region.Id != order.Src {
			return errors.New(fmt.Sprintf("army region %v doesn't match src %v", army.Region.Id, order.Src))
		}
		// validate destination region
		_, ok := self.regions[order.Dst]
		if !ok {
			return errors.New(fmt.Sprintf("invalid destination id %v", order.Dst))
		}
		for _, edge := range army.Region.Edges {
			if edge.Dst.Id == order.Dst {
				continue outerloop
			}
		}
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

func copyArmies(dst, src Armies) error {
	for armyId, army := range src {
		dst[armyId] = newArmy(army)
	}
	return nil
}

var SampleMarchOrder = MarchOrder{
	ArmyOrder: newArmyOrder("army1"),
	Src:       "region3cost",
	Dst:       "region2cost",
}

var SampleAttackOrder = MarchOrder{
	ArmyOrder: newArmyOrder("army2"),
	Src:       "region1",
	Dst:       "region2cost",
}

// Terrain movements are on orders of 10 from 10-100.
// The same goes with boundary (river, wall) movemnt penalties
// army movements modifiers range 1-9.
// this is so army modifiers only break terrain movement ties.
var ExampleTerrainPenalty string = `
    [PLAIN.MOUNTAIN]
    movementPenalty = 30
    [MOUNTAIN.PLAIN]
    movementPenalty = 10
    [PLAIN.PLAIN]
    movementPenalty = 0
    [MOUNTAIN.MOUNTAIN]
    movementPenalty  = 50
    `
