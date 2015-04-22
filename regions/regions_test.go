package regions

import (
	"github.com/BurntSushi/toml"
	"testing"
)

var configRegions string = `
    [region1]
    size = 3
    neighbors = ["region2","region4"]


    [region2]
    size = 3
    terrain = "mountain"
    neighbors = ["region1","region3"]


    [region3]
    size = 3
    neighbors = ["region2","region6"]

    [region4]
    size = 3
    terrain = "plain"
    neighbors = ["region1","region5"]


    [region5]
    size = 3
    neighbors = ["region4"]

    [region6]
    size = 3
    neighbors = ["region3"]
    `

var configRivers string = `
    [yellowfork]
    name="yellow fork"
    borders = ["region2","region1","region1","region4"]
    `

func TestRegions(t *testing.T) {
	var regions Regions
	if _, err := toml.Decode(configRegions, &regions); err != nil {
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

}

func GetRivers() (Rivers, error) {
	var rivers Rivers
	if _, err := toml.Decode(configRivers, &rivers); err != nil {
		return nil, err
	}
	return rivers, nil
}
