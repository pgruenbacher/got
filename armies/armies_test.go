package armies

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/pgruenbacher/got/diplomats"
	"github.com/pgruenbacher/got/families"
	"github.com/pgruenbacher/got/regions"
)

func TestArmies(t *testing.T) {

	var rs regions.Regions
	if _, err := toml.Decode(regions.ExampleRegions, &rs); err != nil {
		t.Error(err)
	}
	if err := rs.ConnectAll(); err != nil {
		t.Error(err)
	}
	var armies Armies
	if _, err := toml.Decode(SampleArmies, &armies); err != nil {
		t.Error(err)
		return
	}

	var table diplomats.DiplomatsTable
	if _, err := toml.Decode(diplomats.ExampleTable, &table); err != nil {
		t.Error(err)
	}
	var h families.Houses
	if _, err := toml.Decode(families.ExampleHouses, &h); err != nil {
		t.Error(err)
		return
	}

	h.InitializeAll()
	if err := table.Init(h); err != nil {
		t.Error(err)
		return
	}
	if err := armies.Init(rs); err != nil {
		t.Error(err)
		return
	}
	// Army Manager
	var armyManager ArmiesManager
	if _, err := toml.Decode(ExampleTerrainPenalty, &armyManager.Config.TerrainPenalties); err != nil {
		t.Error(err)
		return
	}
	if _, err := toml.Decode(ExampleModifiers, &armyManager.Config); err != nil {
		t.Error(err)
		return
	}
	t.Log(armyManager.Config)
	if err := armyManager.Init(armies, rs, table); err != nil {
		t.Error(err)
	}

	// orders := armyManager.GivePossibleOrders("army1")
	// t.Log(orders)

	orders := []MarchOrder{
		SampleMarchOrder,
		SampleMarchOrder2,
	}

	if _, err := armyManager.ReadOrders(orders); err != nil {
		t.Error(err)
	}
}
