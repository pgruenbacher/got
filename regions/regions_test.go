package regions

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestRegions(t *testing.T) {
	var regions Regions
	if _, err := toml.Decode(ExampleRegions, &regions); err != nil {
		t.Error(err)
	}
	if err := regions.ConnectAll(); err != nil {
		t.Error(err)
	}

	rivers, _ := GetRivers()
	for _, boundary := range rivers {
		if err := regions.IncorporateBoundary(boundary); err != nil {
			t.Error(err)
		}

	}
	a, b := regions.Djikstra("region1", "region6", sampleWeightFilter)
	t.Log("cameFrom", a)
	t.Log("cost", b)

}

func sampleWeightFilter(a, b *Region) int {
	switch b.Terrain {
	case Plain:
		return 1
	case Mountain:
		return 3
	default:
		return 1

	}
}

func GetRivers() (Rivers, error) {
	var rivers Rivers
	if _, err := toml.Decode(ExampleRivers, &rivers); err != nil {
		return nil, err
	}
	return rivers, nil
}
