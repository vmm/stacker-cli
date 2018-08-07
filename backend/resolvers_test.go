package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eyeamera/stacker-cli/stacker"
)

func TestResolveFile(t *testing.T) {

	cases := []struct {
		key   string
		param interface{}
		stack stacker.Stack

		expected stacker.StackParam
		errored  bool
	}{
		{
			"foo", "../test/data.txt", &stack{},
			&stackParam{key: "foo", value: "101010"}, false,
		},
		{
			"foo", "../test/doesnotexist.txt", &stack{},
			nil, true,
		},
	}

	for _, c := range cases {
		r, err := ResolveFile(c.key, c.param, c.stack)

		assert.Equal(t, c.expected, r)

		if c.errored {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}

}
