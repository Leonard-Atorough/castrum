package core

import (
	"github.com/leonard-atorough/castrum/ecs"
)

type testComponentA struct {
	value int
}

func (t testComponentA) Name() string         { return "testComponentA" }
func (t testComponentA) Clone() ecs.Component { return testComponentA{value: t.value} }

type testComponentB struct {
	value int
}

func (t testComponentB) Name() string         { return "testComponentB" }
func (t testComponentB) Clone() ecs.Component { return testComponentB{value: t.value} }
