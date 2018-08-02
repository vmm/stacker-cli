package backend

import (
	"fmt"

	"github.com/eyeamera/stacker-cli/stacker"
)

type stack struct {
	name          string
	region        string
	capabilities  string
	templateBody  string
	rawParameters RawParams
	resolver      ParamsResolver
}

func (s *stack) Name() string         { return s.name }
func (s *stack) TemplateBody() string { return s.templateBody }
func (s *stack) Capabilities() []string {
	if s.capabilities != "" {
		return []string{s.capabilities}
	}
	return nil
}
func (s *stack) Region() string { return s.region }
func (s *stack) Params() ([]stacker.StackParam, error) {
	return s.resolver.Resolve(s.rawParameters, s)
}

type fetcher struct {
	cs ConfigStore
	ts TemplateStore
	r  ParamsResolver
}

func newFetcher(cs ConfigStore, ts TemplateStore, r ParamsResolver) *fetcher {
	return &fetcher{cs, ts, r}
}

func (f *fetcher) FetchAll() ([]stacker.Stack, error) {
	stackConfigs, err := f.cs.FetchAll()
	if err != nil {
		return []stacker.Stack{}, fmt.Errorf("unable to fetch stacks: %s", err)
	}

	return f.fetchTemplates(stackConfigs)
}

func (f *fetcher) Fetch(name string) ([]stacker.Stack, error) {
	stackConfigs, err := f.cs.Fetch(name)
	if err != nil {
		return []stacker.Stack{}, fmt.Errorf("unable to fetch stack %s: %s", name, err)
	}

	return f.fetchTemplates(stackConfigs)
}

// Fetch the templates for each stack to get a final list of params,
// and the template body
func (f *fetcher) fetchTemplates(stackConfigs []stackConfig) ([]stacker.Stack, error) {
	stacks := make([]stacker.Stack, 0)
	for _, stackConfig := range stackConfigs {
		t, err := f.ts.Fetch(stackConfig.TemplateName)
		if err != nil {
			return stacks, fmt.Errorf("unable to fetch template %s: %s", stackConfig.TemplateName, err)
		}

		rp := make(RawParams)
		for _, k := range t.Parameters() {
			if v, ok := stackConfig.Parameters[k]; ok {
				rp[k] = v
			}
		}

		s := &stack{
			name:          stackConfig.Name,
			region:        stackConfig.Region,
			capabilities:  stackConfig.Capabilities,
			templateBody:  t.Body(),
			rawParameters: rp,
			resolver:      f.r,
		}

		stacks = append(stacks, s)
	}

	return stacks, nil
}
