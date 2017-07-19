package backend

import (
	"testing"

	"github.com/eyeamera/stacker-cli/client"
	"github.com/stretchr/testify/assert"
)

func TestResolveFile(t *testing.T) {

	cases := []struct {
		key   string
		param interface{}
		stack StackInfo

		expected client.StackParam
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
