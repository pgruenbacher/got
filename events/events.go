package events

import "github.com/pgruenbacher/got/utils"

type Event struct {
	Id string
}

func NewEvent() Event {
	return Event{
		Id: utils.RandSeq(6),
	}
}

type EventsInterface interface{}
