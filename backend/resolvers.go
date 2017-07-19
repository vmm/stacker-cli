package backend

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/eyeamera/stacker-cli/client"
	"github.com/pkg/errors"
)

// ResolveStackOutput provides a mechanism to lookup an output from a stack within
// the same region
//
// Example Usage:
//
//   parameters:
//     VpcID:
//       Stack: Foo-VPC.VpcID
//
// where 'Foo-VPC' is the stack name, and 'VpcId' is the stack output
func ResolveStackOutput(key string, param interface{}, stack StackInfo) (client.StackParam, error) {
	s := strings.SplitN(fmt.Sprint(param), ".", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("expected to receive input in format <stack>.<output>")
	}

	stackName, outputName := s[0], s[1]
	client := client.New(client.NewCloudformationClient(stack.Region()))
	si, err := client.Get(stackName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch stack `%s`", stackName)
	}

	for _, o := range si.Outputs {
		if o.Key == outputName {
			return &stackParam{key: key, value: o.Value}, nil
		}
	}

	return nil, fmt.Errorf("unable to find output `%s` on stack `%s`", outputName, stackName)
}

func ResolveFile(key string, param interface{}, stack StackInfo) (client.StackParam, error) {
	path := fmt.Sprint(param)
	r, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open file %s", path)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read file %s", path)
	}

	return &stackParam{key: key, value: strings.TrimSuffix(string(b), "\n")}, nil
}
