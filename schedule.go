package main

import (
	"time"
)

type class struct {
	starts time.Time
	ends time.Time
	name string
}

type schoolDay []class
type schedule map[string]schoolDay

//func addClass()