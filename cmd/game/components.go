package main

import "github.com/leonard-atorough/castrum/types"

// Velocity represents linear motion in units per second.
type Velocity struct {
	Linear types.Vector2
}

// Player is a marker component for entities controlled by player input.
type Player struct{}
