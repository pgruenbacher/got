package families

type House struct {
	Id   HouseId
	Name string
}
type HouseId string

type Houses map[HouseId]*House

func (self Houses) InitializeAll() {
	for houseId, house := range self {
		house.Id = houseId
	}
}

var ExampleHouses = `
    [house1]
    name="stark"
    [house2]
    name="lannister"
    [house3]
    name="mormont"
    [house4]
    name="tyrell"
`
