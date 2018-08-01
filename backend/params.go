package backend

import (
	"fmt"
	"reflect"

	"github.com/eyeamera/stacker-cli/client"
	"github.com/pkg/errors"
)

// A subset of client.Stack available for resolvers
type StackInfo interface {
	Region() string
}

// Implements client.StackParam
type stackParam struct {
	key         string
	value       string
	usePrevious bool
}

func (sp *stackParam) Key() string       { return sp.key }
func (sp *stackParam) Value() string     { return sp.value }
func (sp *stackParam) UsePrevious() bool { return sp.usePrevious }

type RawParams map[string]interface{}

type Resolver func(key string, param interface{}, stack StackInfo) (client.StackParam, error)

type ParamsResolver interface {
	Resolve(rp RawParams, stack StackInfo) (client.StackParams, error)
}

type paramsResolver struct {
	resolvers map[string]Resolver
}

func NewParamsResolver() *paramsResolver {
	return &paramsResolver{
		resolvers: make(map[string]Resolver),
	}
}

func (pr *paramsResolver) Add(key string, r Resolver) {
	pr.resolvers[key] = r
}

func (pr *paramsResolver) Resolve(rp RawParams, stack StackInfo) (client.StackParams, error) {
	sp := make(client.StackParams, 0)
	for k, v := range rp {
		r, err := pr.resolve(k, v, stack)
		if err != nil {
			return nil, errors.Wrapf(err, "an error occured resolving %s", k)
		}

		sp = append(sp, r)
	}
	return sp, nil
}

func (pr *paramsResolver) resolve(k string, v interface{}, stack StackInfo) (client.StackParam, error) {
	original := reflect.ValueOf(v)
	switch original.Kind() {
	case reflect.Map:
		keys := original.MapKeys()
		if len(keys) > 1 {
			return nil, errors.New("unexpected map with more than one key")
		}

		key := fmt.Sprint(keys[0])
		resolver, ok := pr.resolvers[key]
		if !ok {
			return nil, fmt.Errorf("unknown resolver `%s`", key)
		}

		return resolver(k, original.MapIndex(keys[0]).Interface(), stack)
	case reflect.Slice:
		var s string
		for i := 0; i < original.Len(); i++ {
			if i > 0 {
				s += ","
			}
			v, err := pr.resolve(k, original.Index(i).Interface(), stack)
			if err != nil {
				return nil, err
			}
			s += v.Value()
		}
		return &stackParam{key: k, value: s}, nil
	}
	return &stackParam{key: k, value: fmt.Sprint(v)}, nil
}
