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
	RETREAT                Context = "RETREAT"
	ATTACK                 Context = "ATTACK"
	REDIRECT_ATTACK        Context = "REDIRECT_ATTACK"
	MARCH                  Context = "MARCH"
	CANCEL_NEUTRAL_PRESENT Context = "CANCEL_NEUTRAL_PRESENT"
	/*
	 *  Event Contexts
	 */
	// retreat as enemy attacks
	STRATEGIC_RETREAT Context = "STRATEGIC_RETREAT"
	// attack, but all enemy marched elsewhere
	ATTACK_PURSUIT Context = "ATTACK_PURSUIT"
	// army that was marching is attacked
	CAUGHT_ATTACK Context = "CAUGHT_ATTACK"
	// Suprise attack for an unexpected enemy in region
	SURPRISE_ATTACK Context = "SURPRISE_ATTACK"
	// Surprise retreat into a region with an unexpected enemy
	SURPRISE_RETREAT Context = "SURPRISE_RETREAT"
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
	Ctx Context
}

// manager structeure
type ArmiesManager struct {
	Armies    Armies
	regions   regions.Regions
	diplomacy diplomats.DiplomatsTable
	Config    Config
}

type Config struct {
	TerrainPenalties  TerrainPenalties
	DefenseBonuses    map[regions.Terrain]CombatModifier `toml:"Defense_Bonuses",validate:"max=1,min=-1"`
	ConstantModifiers map[Context]CombatModifier         `toml:"Context_Modifiers",validate:"max=1,min=-1"`
}

type TerrainPenalties map[regions.Terrain]TerrainPenalty

type TerrainPenalty map[regions.Terrain]Penalty

type Penalty struct {
	MovementPenalty int
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
	boundaryPenalty = edge.Boundary.MovePenalty()
	return terrainPenalty + boundaryPenalty + self.Armies[order.ArmyId].Size
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
				attackOrder := newMarchOrder(army.Id, to.Src.Id, to.Dst.Id, ATTACK)
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

	tmpArmies := make(Armies, len(self.Armies))
	copyArmies(tmpArmies, self.Armies)

	events, battles, err := self.checkDestinations(tmpArmies, orders)
	if err != nil {
		return e, err
	}
	e = append(e, events...)
	fmt.Println(tmpArmies["army1"].Size, tmpArmies["army1"].Morale, "amrmy2:", tmpArmies["army2"].Size, tmpArmies["army2"].Morale)
	_, err = self.resolveBattles(battles)
	fmt.Println(tmpArmies["army1"].Size, tmpArmies["army1"].Morale, "amrmy2:", tmpArmies["army2"].Size, tmpArmies["army2"].Morale)
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

func (self *ArmiesManager) checkDestinations(tmpArmies Armies, orders []MarchOrder) (events []MarchEvent, combats battles, err error) {
	// create a temporary copy of the armies and their future destinations.
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
		army := tmpArmies[order.ArmyId]
		// innerLoop:
		for _, army2 := range armiesWithin(tmpArmies, self.regions[order.Dst]) {
			// army may already be in combat, but continue other possible attack directions if it is returning attack or attacking different army
			if self.diplomacy.IsEnemy(army.House, army2.House) {
				// initiate attack on army within region
				// set status of both armies as in combat
				army.setInCombat()
				army2.setInCombat()
				switch order.Ctx {
				// army attempted to retreat from battle, but is now caught in another battle by an enemy that had moved into that region quicker
				case RETREAT:
					events = append(events, newMarchEvent(order.ArmyId, order.Src, order.Dst, ATTACK))
					fmt.Println("caught retreat!")
					combats = append(combats, newBattle(army, army2, ATTACK))

				case ATTACK:
					// march event with attack refers to a successful intentional attack event
					events = append(events, newMarchEvent(order.ArmyId, order.Src, order.Dst, ATTACK))
					fmt.Println("attack!")
					combats = append(combats, newBattle(army, army2, ATTACK))
				case MARCH:
					// march event with suprise attack refers to an unintentional attack of enemy in region
					events = append(events, newMarchEvent(order.ArmyId, order.Src, order.Dst, SURPRISE_ATTACK))
					fmt.Println("surprise attack!")
					combats = append(combats, newBattle(army, army2, SURPRISE_ATTACK))
				}
				continue outerLoop

			}
			// if army is being attacked, and not marching against an enemy then disregard other march orders
			if army.inCombat() {
				// cancel order
				continue outerLoop
			}
			// army is neither attacking nor being attacked
			if self.diplomacy.IsAlly(army.House, army2.House) {
				// army may move into region if there is an ally and there is no military
				fmt.Println("allies! move in!")
				army.March(army.Region.Edges[order.Dst])
				events = append(events, newMarchEvent(order.ArmyId, order.Src, order.Dst, MARCH))
				continue outerLoop
			}
			// ELSE there is a neutral army present, cannot enter the region, should not have been a legal move in the first place.
			// since there is no idea if the neutral army will be staying or leaving to the player if it had been there in the first place.
			// if quicker netural army moved there first...then tough luck.
			// cancel army order
			fmt.Println("cancel order")
			events = append(events, newMarchEvent(order.ArmyId, order.Src, order.Dst, CANCEL_NEUTRAL_PRESENT))
			continue outerLoop
		}
		// if army entering foreign region but not at war and does not have permission...

		// if no armies present, then move on in!
		army.March(army.Region.Edges[order.Dst])
		events = append(events, newMarchEvent(order.ArmyId, order.Src, order.Dst, MARCH))
		fmt.Println("move army!")
	}
	return events, combats, nil
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

func newMarchOrder(id armyId, src, dst regions.RegionId, ctx Context) MarchOrder {
	return MarchOrder{
		ArmyOrder: newArmyOrder(id),
		Src:       src,
		Dst:       dst,
		Ctx:       ctx,
	}
}

func newMarchEvent(id armyId, src, dst regions.RegionId, ctx Context) MarchEvent {
	return MarchEvent{
		ArmyEvent: newArmyEvent(id),
		Src:       src,
		Dst:       dst,
		Ctx:       ctx,
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
	Ctx:       MARCH,
}

var SampleMarchOrder2 = MarchOrder{
	ArmyOrder: newArmyOrder("army2"),
	Src:       "region1",
	Dst:       "region2cost",
	Ctx:       MARCH,
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

var ExampleModifiers string = `
	[Defense_Bonuses]
	PLAIN = 0.0
	HILL = 0.1
	MOUNTAIN = 0.3
	[Context_Modifiers]
	SURPRISE_ATTACK=-0.3
	`
