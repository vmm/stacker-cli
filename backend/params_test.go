package backend

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eyeamera/stacker-cli/stacker"
)

func TestParamsResolverResolve(t *testing.T) {
	pr := NewParamsResolver()

	rp := RawParams{
		"str":     "bar",
		"int":     123,
		"bool":    true,
		"float":   1.05,
		"arr":     []string{"a", "b", "c"},
		"arrdeep": []interface{}{"a", []interface{}{1, "2", 3.14}, false},
	}

	expected := []stacker.StackParam{
		&stackParam{key: "str", value: "bar"},
		&stackParam{key: "int", value: "123"},
		&stackParam{key: "bool", value: "true"},
		&stackParam{key: "float", value: "1.05"},
		&stackParam{key: "arr", value: "a,b,c"},
		&stackParam{key: "arrdeep", value: "a,1,2,3.14,false"},
	}

	sps, err := pr.Resolve(rp, &stack{})

	assert.Nil(t, err)
	assert.ElementsMatch(t, sps, expected)
}

func TestParamsResolverCustomResolve(t *testing.T) {
	greet := func(key string, param interface{}, stack stacker.Stack) (stacker.StackParam, error) {
		name := fmt.Sprint(param)
		return &stackParam{
			key:   key,
			value: fmt.Sprintf("Hello, %s", name),
		}, nil
	}

	pr := NewParamsResolver()
	pr.Add("greet", greet)

	rp := RawParams{
		"param1": map[string]string{
			"greet": "paul",
		},
	}

	expected := []stacker.StackParam{
		&stackParam{key: "param1", value: "Hello, paul"},
	}

	sps, err := pr.Resolve(rp, &stack{})

	assert.Nil(t, err)
	assert.ElementsMatch(t, expected, sps)
}

func TestParamsResolverMultipleCustomResolve(t *testing.T) {
	greet := func(key string, param interface{}, stack stacker.Stack) (stacker.StackParam, error) {
		name := fmt.Sprint(param)
		return &stackParam{
			key:   key,
			value: fmt.Sprintf("Hello, %s", name),
		}, nil
	}

	pr := NewParamsResolver()
	pr.Add("greet", greet)

	rp := RawParams{
		"param1": []map[string]string{
			{"greet": "alice"},
			{"greet": "bob"},
		},
	}

	expected := []stacker.StackParam{
		&stackParam{key: "param1", value: "Hello, alice,Hello, bob"},
	}

	sps, err := pr.Resolve(rp, &stack{})

	assert.Nil(t, err)
	assert.ElementsMatch(t, expected, sps)
}
