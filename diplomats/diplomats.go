package diplomats

import "github.com/pgruenbacher/got/families"

type Proposal struct {
	from families.HouseId
	to   families.HouseId
}

type PeaceProposal struct {
	Proposal
}

type DiplomatsTable struct {
	Starting_relations map[families.HouseId]Relations `toml:"relations"`

	// only one relation may exist between each house.
	RelationsTable map[families.HouseId]Relations

	houses families.Houses
	// pending proposals that are collected by orders and then forwarded to players on next turn
	proposals []Proposal
}

type Relations map[families.HouseId]*Relation
type OfficialStatus string
type RelationStatus string

const (
	ALLIED  OfficialStatus = "ALLIED"
	ENEMY   OfficialStatus = "ENEMY"
	NEUTRAL OfficialStatus = "NEUTRAL"

	FRIENDLY RelationStatus = "FRIENDLY"
	HATRED   RelationStatus = "HATRED"
	UNKNOWN  RelationStatus = "UNKNOWN"
)

func (self *DiplomatsTable) IsEnemy(houseId1, houseId2 families.HouseId) bool {
	return self.RelationsTable[houseId1][houseId2].OfficialStatus == ENEMY
}

func (self *DiplomatsTable) IsAlly(a, b families.HouseId) bool {
	return self.RelationsTable[a][b].OfficialStatus == ALLIED
}

type Relation struct {
	house1         *families.House
	house2         *families.House
	OfficialStatus OfficialStatus `toml:"official_status"`
	RelationStatus RelationStatus `toml:"relation_status"`
}

func (self *DiplomatsTable) Init(h families.Houses) error {
	self.houses = h
	err := self.initalizeRelations()
	return err
}

// create the relations table from the starting relations.
// each house should point to a relation that is shared by one other house
func (self *DiplomatsTable) initalizeRelations() error {
	self.makeRelations()
	for h1, relations := range self.Starting_relations {
		for h2, relation := range relations {
			self.RelationsTable[h1][h2].OfficialStatus = relation.OfficialStatus
			self.RelationsTable[h1][h2].RelationStatus = relation.RelationStatus
		}
	}
	return nil
}

func (self *DiplomatsTable) makeRelations() {
	self.RelationsTable = make(map[families.HouseId]Relations, len(self.houses))
	for _, house := range self.houses {
		self.RelationsTable[house.Id] = make(Relations, len(self.houses)-1)
	}
	// start with first house, make relation for each other house
	for houseId1, house1 := range self.houses {
		for houseId2, house2 := range self.houses {
			if houseId2 == houseId1 {
				continue
			}
			if self.alreadyExists(house1.Id, house2.Id) {
				continue
			}

			relation := newRelation(house1, house2)
			self.RelationsTable[house1.Id][house2.Id] = relation
			self.RelationsTable[house2.Id][house1.Id] = relation
		}
	}
}

// wil check if h2 House2 already points to a relation that points to h1 House1
func (self *DiplomatsTable) alreadyExists(h1, h2 families.HouseId) bool {
	if rels, ok := self.RelationsTable[h2]; ok {
		for _, relation := range rels {
			// check if relation refers to both houses
			if relation.house1.Id == h1 {
				if relation.house2.Id == h2 {
					return true
				}
			}
			if relation.house1.Id == h2 {
				if relation.house2.Id == h1 {
					return true
				}
			}
		}
	}
	return false
}

/*
 * Constructors
 *
 */
func newRelation(h1, h2 *families.House) *Relation {
	return &Relation{
		house1:         h1,
		house2:         h2,
		OfficialStatus: NEUTRAL,
		RelationStatus: UNKNOWN,
	}
}

var ExampleTable = `
    
    [relations.house1.house2]
    official_status="ENEMY"
    relation_status="HATRED"  
    `
