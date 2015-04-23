package actions

import "github.com/pgruenbacher/got/utils"

type orderId string

type Order struct {
	Id string
}

func NewOrder() Order {
	return Order{
		Id: utils.RandSeq(7),
	}
}

type OrderInterface interface {
}
