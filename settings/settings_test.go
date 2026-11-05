package settings_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Leonard-Atorough/castrum/settings"
)

type positiveValidator struct{}

func (positiveValidator) Validate(v int) error {
	if v < 0 {
		return fmt.Errorf("negative")
	}
	return nil
}

type slicePolicy struct {
	validated int
}

func (p *slicePolicy) Validate(value []int) error {
	p.validated++
	if len(value) == 0 || value[0] < 0 {
		return fmt.Errorf("invalid values")
	}
	return nil
}

func (slicePolicy) Clone(value []int) []int {
	return append([]int(nil), value...)
}

func TestGroupValuesAndValidation(t *testing.T) {
	policy := settings.Policy[int]{Validator: positiveValidator{}}
	if _, err := settings.New(-1, policy); err == nil {
		t.Fatal("invalid initial value accepted")
	}
	group, err := settings.New(1, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := group.Set(2); err != nil {
		t.Fatal(err)
	}
	if value, err := group.Value(); err != nil || value != 2 {
		t.Fatalf("value = %d, error %v", value, err)
	}
	other, err := settings.New(2, policy)
	if err != nil {
		t.Fatal(err)
	}
	*group = *other
	copied := *group
	if err := copied.Set(3); err != nil {
		t.Fatal(err)
	}
	if value, _ := group.Value(); value != 3 {
		t.Fatal("copied group did not share state")
	}
	var zero settings.Group[int]
	if _, err := zero.Value(); err == nil {
		t.Fatal("zero group returned value")
	}
	if err := zero.Set(1); err == nil {
		t.Fatal("zero group accepted update")
	}
}

func TestSetReplacesEqualValues(t *testing.T) {
	first, second := 1, 1
	group, err := settings.New(&first, settings.Policy[*int]{})
	if err != nil {
		t.Fatal(err)
	}
	if err := group.Set(&second); err != nil {
		t.Fatal(err)
	}
	if got, err := group.Value(); err != nil || got != &second {
		t.Fatalf("replacement = %p, error %v", got, err)
	}
}

func TestGroupPolicyValidationAndCloning(t *testing.T) {
	validator := &slicePolicy{}
	policy := settings.Policy[[]int]{Validator: validator, Cloner: slicePolicy{}}
	initial := []int{1, 2}
	group, err := settings.New(initial, policy)
	if err != nil {
		t.Fatal(err)
	}
	initial[0] = 9
	snapshot, err := group.Value()
	if err != nil {
		t.Fatal(err)
	}
	snapshot[1] = 9
	if got, _ := group.Value(); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("initial or snapshot aliases group: %v", got)
	}
	if err := group.Set([]int{-1}); err == nil {
		t.Fatal("invalid update accepted")
	}
	if got, _ := group.Value(); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("invalid update changed value: %v", got)
	}
	replacement := []int{3, 4}
	if err := group.Set(replacement); err != nil {
		t.Fatal(err)
	}
	replacement[0] = 9
	if got, _ := group.Value(); !reflect.DeepEqual(got, []int{3, 4}) {
		t.Fatalf("replacement aliases group: %v", got)
	}
	if validator.validated != 3 {
		t.Fatalf("validation calls = %d, want 3", validator.validated)
	}
}

func TestGroupUpdate(t *testing.T) {
	group, err := settings.New(1, settings.Policy[int]{Validator: positiveValidator{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := group.Update(func(v int) int { return v }); err != nil {
		t.Fatal(err)
	}
	if got, err := group.Value(); err != nil || got != 1 {
		t.Fatalf("unchanged update: value %d, error %v", got, err)
	}
	if err := group.Update(func(v int) int { return -1 }); err == nil {
		t.Fatal("invalid update accepted")
	}
	if value, _ := group.Value(); value != 1 {
		t.Fatalf("invalid update changed value to %d", value)
	}
	if err := group.Update(func(v int) int { return v + 1 }); err != nil {
		t.Fatal(err)
	}
	if value, _ := group.Value(); value != 2 {
		t.Fatalf("updated value = %d", value)
	}
	if err := group.Update(nil); err == nil {
		t.Fatal("nil callback accepted")
	}
	var zero settings.Group[int]
	if err := zero.Update(func(v int) int { return v }); err == nil {
		t.Fatal("uninitialized group accepted update")
	}
}

func TestGroupUpdateRejectsNestedMutations(t *testing.T) {
	group, err := settings.New(1, settings.Policy[int]{})
	if err != nil {
		t.Fatal(err)
	}
	copy := *group
	err = group.Update(func(v int) int {
		if err := copy.Set(2); err == nil {
			t.Error("nested Set accepted")
		}
		if err := copy.Update(func(v int) int { return v + 1 }); err == nil {
			t.Error("nested Update accepted")
		}
		return v
	})
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := group.Value(); value != 1 {
		t.Fatalf("nested mutation changed value to %d", value)
	}
	if err := group.Set(3); err != nil {
		t.Fatal(err)
	}
	other, _ := settings.New(4, settings.Policy[int]{})
	if err := group.Update(func(v int) int { *group = *other; return v }); err == nil {
		t.Fatal("replacement during update accepted")
	}
}

func TestGroupUpdateClonesMutableValues(t *testing.T) {
	group, err := settings.New([]int{1, 2}, settings.Policy[[]int]{Validator: &slicePolicy{}, Cloner: slicePolicy{}})
	if err != nil {
		t.Fatal(err)
	}
	var callbackValue []int
	if err := group.Update(func(v []int) []int {
		callbackValue = v
		v[0] = 3
		return v
	}); err != nil {
		t.Fatal(err)
	}
	callbackValue[0] = 9
	if got, _ := group.Value(); !reflect.DeepEqual(got, []int{3, 2}) {
		t.Fatalf("callback retained group data: %v", got)
	}
	if err := group.Update(func(v []int) []int { v[0] = -1; return v }); err == nil {
		t.Fatal("invalid update accepted")
	}
	if got, _ := group.Value(); !reflect.DeepEqual(got, []int{3, 2}) {
		t.Fatalf("invalid update changed group: %v", got)
	}
}

func TestFunctionPolicies(t *testing.T) {
	policy := settings.Policy[[]int]{
		Validator: settings.ValidatorFunc[[]int](func(v []int) error {
			if len(v) == 0 {
				return fmt.Errorf("empty")
			}
			return nil
		}),
		Cloner: settings.ClonerFunc[[]int](func(v []int) []int { return append([]int(nil), v...) }),
	}
	initial := []int{1}
	group, err := settings.New(initial, policy)
	if err != nil {
		t.Fatal(err)
	}
	initial[0] = 2
	if value, _ := group.Value(); value[0] != 1 {
		t.Fatal("function cloner retained caller data")
	}
	if err := group.Set(nil); err == nil {
		t.Fatal("function validator accepted invalid value")
	}
	var validator settings.ValidatorFunc[int]
	if _, err := settings.New(1, settings.Policy[int]{Validator: validator}); err == nil {
		t.Fatal("nil validator function accepted")
	}
	var cloner settings.ClonerFunc[int]
	if _, err := settings.New(1, settings.Policy[int]{Cloner: cloner}); err == nil {
		t.Fatal("nil cloner function accepted")
	}
}

func TestPolicyRejectsNestedMutations(t *testing.T) {
	var group *settings.Group[int]
	var setErr, updateErr error
	attemptMutation := false
	policy := settings.Policy[int]{Validator: settings.ValidatorFunc[int](func(value int) error {
		if attemptMutation {
			attemptMutation = false
			copy := *group
			setErr = copy.Set(2)
			updateErr = copy.Update(func(int) int { return 3 })
			return fmt.Errorf("invalid")
		}
		return nil
	})}
	var err error
	group, err = settings.New(1, policy)
	if err != nil {
		t.Fatal(err)
	}
	attemptMutation = true
	if err := group.Set(4); err == nil {
		t.Fatal("invalid value accepted")
	}
	if setErr == nil || updateErr == nil {
		t.Fatalf("nested mutation errors: Set %v, Update %v", setErr, updateErr)
	}
	if value, err := group.Value(); err != nil || value != 1 {
		t.Fatalf("invalid update changed value: %d, error %v", value, err)
	}
	if err := group.Set(5); err != nil {
		t.Fatalf("guard remained set after validation: %v", err)
	}
}

func TestValueRejectsMutationFromCloner(t *testing.T) {
	var group *settings.Group[int]
	var nestedErr error
	attemptMutation := false
	policy := settings.Policy[int]{Cloner: settings.ClonerFunc[int](func(value int) int {
		if attemptMutation {
			attemptMutation = false
			nestedErr = group.Set(2)
		}
		return value
	})}
	var err error
	group, err = settings.New(1, policy)
	if err != nil {
		t.Fatal(err)
	}
	attemptMutation = true
	if value, err := group.Value(); err != nil || value != 1 {
		t.Fatalf("value = %d, error %v", value, err)
	}
	if nestedErr == nil {
		t.Fatal("cloner mutated the group while copying")
	}
	if value, err := group.Value(); err != nil || value != 1 {
		t.Fatalf("current value = %d, error %v", value, err)
	}
	if err := group.Set(2); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyGuardClearsAfterPanic(t *testing.T) {
	panicClone := false
	group, err := settings.New(1, settings.Policy[int]{Cloner: settings.ClonerFunc[int](func(value int) int {
		if panicClone {
			panic("cloner failed")
		}
		return value
	})})
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected cloner panic")
			}
		}()
		panicClone = true
		_, _ = group.Value()
	}()
	panicClone = false
	if err := group.Set(2); err != nil {
		t.Fatalf("guard remained set after panic: %v", err)
	}
}

func TestPolicyRejectsGroupReplacement(t *testing.T) {
	for _, operation := range []string{"value", "set", "update", "validate"} {
		t.Run(operation, func(t *testing.T) {
			other, err := settings.New(9, settings.Policy[int]{})
			if err != nil {
				t.Fatal(err)
			}
			var group *settings.Group[int]
			replace := false
			policy := settings.Policy[int]{
				Cloner: settings.ClonerFunc[int](func(value int) int {
					if replace && operation != "validate" {
						*group = *other
					}
					return value
				}),
				Validator: settings.ValidatorFunc[int](func(int) error {
					if replace && operation == "validate" {
						*group = *other
					}
					return nil
				}),
			}
			group, err = settings.New(1, policy)
			if err != nil {
				t.Fatal(err)
			}
			replace = true
			switch operation {
			case "value":
				_, err = group.Value()
			case "set", "validate":
				err = group.Set(2)
			case "update":
				err = group.Update(func(value int) int { return value + 1 })
			}
			if err == nil {
				t.Fatal("group replacement during policy accepted")
			}
			if value, err := group.Value(); err != nil || value != 9 {
				t.Fatalf("replacement side effect: value %d, error %v", value, err)
			}
		})
	}
}
