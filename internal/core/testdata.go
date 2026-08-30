package core

type testComponentA struct {
	value int
}

func (t testComponentA) Name() string     { return "testComponentA" }
func (t testComponentA) Clone() Component { return testComponentA{value: t.value} }

type testComponentB struct {
	value int
}

func (t testComponentB) Name() string     { return "testComponentB" }
func (t testComponentB) Clone() Component { return testComponentB{value: t.value} }
