package armies

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type battle struct {
	army1 *Army
	army2 *Army
	ctx   Context
}

type battles []battle

// +gen stringer
type combatStatus int

// +gen stringer
type defenseStatus int

type CombatModifier float32

const (
	// combat phases simply indicate the initial skirmish, usually apply
	COMBAT_PHASE1 combatStatus = 1 + iota
	COMBAT_PHASE2

	// Likewise the defense status simply indicates the number of turns the army has
	// spent using defening turns
	DEFENDED_PHASE1 defenseStatus = 1 + iota
	DEFENDED_PHASE2
)

func (self *Army) setInCombat() {
	if self.combatState == 0 {
		self.combatState++
	}
}

func (self Army) inCombat() bool {
	if self.combatState == 0 {
		return false
	}
	return true
}

func (self Army) defending() bool {
	return self.DefenseState != 0
}

func (self *Army) setDefense() {
	if self.DefenseState == 0 {
		self.DefenseState++
	}
}

func (self ArmiesManager) defenseBonus(army *Army) CombatModifier {
	return self.Config.DefenseBonuses[army.Region.Terrain]
}

type CombatContext string

const (
	ROUTED    CombatContext = "ROUTE"
	DESTROYED CombatContext = "DESTRUCTION"
	DEFEATED  CombatContext = "DEFEATED"
	DRAW      CombatContext = "DRAW"
)

type combatEvent struct {
	TargetArmy armyId
	ByArmy     armyId
	Ctx        CombatContext
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func (self ArmiesManager) resolveBattles(battles battles) (events []combatEvent, err error) {
	tmpAttackMap := make(map[armyId]armyId)
	// set the combat mofidifiers which will be accumulated
	var bonus1, bonus2 CombatModifier
	// validation section, and mapping attacks
	for _, battle := range battles {
		if _, ok := tmpAttackMap[battle.army1.Id]; !ok {
			// army 1 is attackign army 2
			tmpAttackMap[battle.army1.Id] = battle.army2.Id
		} else {
			return events, errors.New(fmt.Sprintf("there cannot be multiple attack battles for army %v", battle.army1.Id))
		}
	}

	for _, battle := range battles {
		// if army1 is attacking army2
		if battle.army1.defending() && !battle.army2.defending() {
			// army 2 is attacking defending army 1
			bonus1 = bonus1 + self.defenseBonus(battle.army1)
			bonus2 = bonus2 + CombatModifier(battle.army2.Region.Edges[battle.army1.Region.Id].Boundary.AttackPenalty())
			event := inflictDamages(battle.army1, battle.army2, bonus1, bonus2)
			events = append(events, event)
			// add defense bonus

		} else if !battle.army1.defending() && battle.army2.defending() {
			// army 1 is attacking defended army 2
			bonus1 = bonus1 + CombatModifier(battle.army1.Region.Edges[battle.army2.Region.Id].Boundary.AttackPenalty())
			bonus2 = bonus2 + self.defenseBonus(battle.army2)
			event := inflictDamages(battle.army1, battle.army2, bonus1, bonus2)
			events = append(events, event)
			// add defense bonuis

		} else if attackingEachother(tmpAttackMap, battle) {
			// armies 1 and 2 are attacking eachother, need to cancel the second battle
			event := inflictDamages(battle.army1, battle.army2, bonus1, bonus2)
			events = append(events, event)

		} else if battle.ctx == SURPRISE_ATTACK {
			// else army 1 has been surprised by army 2, use the SUPRISE ATTACK MODIFIER on army 1
			bonus1 = bonus1 + self.Config.ConstantModifiers[SURPRISE_ATTACK]
			event := inflictDamages(battle.army1, battle.army2, bonus1, bonus2)
			events = append(events, event)
			// perform combat
		}
	}
	return events, nil
}

func attackingEachother(tmpMap map[armyId]armyId, battle battle) bool {
	return tmpMap[battle.army1.Id] == battle.army2.Id && tmpMap[battle.army2.Id] == battle.army1.Id
}

// weights can be terrain penalties, army size differences, etc. Must be great
// army quality stays constant during battles
func inflictDamages(army1, army2 *Army, weight1, weight2 CombatModifier) combatEvent {
	// damage from each army is a product of the army size and quality. therefore quality enhances the initial damage value
	inflict1 := army1.Size * army1.Quality
	inflict2 := army2.Size * army2.Quality
	// a reasonable fraction is taken, to improve pacing of battles to last
	// a random modifier is included
	randSeed1 := rand.Float32()
	randSeed2 := rand.Float32()
	// random infliction algorithm! so that 3-4 battles between similar sized adversaries will likely result in one destruction
	inflict1 = inflict1/6 + int(float32(inflict1)/6*randSeed1)
	inflict2 = inflict2/6 + int(float32(inflict2)/6*randSeed2)

	// apply the weights
	inflict1 = inflict1 + int(CombatModifier(inflict1)*weight1)
	inflict2 = inflict2 + int(CombatModifier(inflict2)*weight2)

	// now inflict the damages upon each army size, relative to their quality
	// compare the damage of the battle, this will result in a victor and loser of the round
	// morale will drop for the loser. morale can't be regained during battles.
	// if morale drops to zero, then return a routing event.

	// else do nothing to morale if it is a tie

	// perform the damages
	// check for total destruction of an army
	damage1 := inflict2 / army1.Quality
	damage2 := inflict1 / army2.Quality
	if damage1 > army1.Size {
		// total destruction of army1
		army1.Size = 0
		army2.Size = army2.Size - damage2
		return newCombatEvent(army1.Id, army2.Id, DESTROYED)
	} else if damage2 > army2.Size {
		// total destruction of army 2
		army2.Size = 0
		army1.Size = army1.Size - damage1
		return newCombatEvent(army2.Id, army1.Id, DESTROYED)
	} else {
		// army size decreases for both armies
		army1.Size = army1.Size - damage1
		army2.Size = army2.Size - damage2

		if inflict1 > inflict2 {
			army2.Morale = army2.Morale - 1
			if army2.Morale <= 0 {
				// army 2 routes
				return newCombatEvent(army2.Id, army1.Id, ROUTED)
			} else {
				// army 2 defeated in skirmish
				return newCombatEvent(army2.Id, army1.Id, DEFEATED)
			}
		} else if inflict2 > inflict1 {
			army1.Morale = army1.Morale - 1
			if army1.Morale <= 0 {
				// army 1 routes
				return newCombatEvent(army1.Id, army2.Id, ROUTED)
			} else {
				// army 1 defeated in skirmish
				return newCombatEvent(army1.Id, army2.Id, DEFEATED)
			}
		}

	}
	// else then simply perform a tie
	return newCombatEvent(army1.Id, army2.Id, DRAW)

}

func newCombatEvent(armyId1, armyId2 armyId, ctx CombatContext) combatEvent {
	return combatEvent{
		armyId1,
		armyId2,
		ctx,
	}
}

func newBattle(army1, army2 *Army, context Context) battle {
	return battle{
		army1,
		army2,
		context,
	}
}
