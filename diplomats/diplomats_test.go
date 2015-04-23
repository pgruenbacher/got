package diplomats

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/pgruenbacher/got/families"
)

func TestTable(t *testing.T) {
	var table DiplomatsTable
	if _, err := toml.Decode(ExampleTable, &table); err != nil {
		t.Error(err)
	}
	var h families.Houses
	if _, err := toml.Decode(families.ExampleHouses, &h); err != nil {
		t.Error(err)
	}
	h.InitializeAll()
	table.houses = h
	table.initalizeRelations()

	if table.RelationsTable["house1"]["house2"] != table.RelationsTable["house2"]["house1"] {
		t.Error("unequal relations")
	}
	if table.RelationsTable["house1"]["house2"] == table.RelationsTable["house2"]["house3"] {
		t.Error("inccorect equal relations")
	}
}
