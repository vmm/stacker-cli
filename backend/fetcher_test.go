package backend

import (
	"testing"

	"github.com/eyeamera/stacker-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fakeConfigStore struct {
	mock.Mock
}

func (f *fakeConfigStore) Fetch(name string) ([]stackConfig, error) {
	r := f.Called(name)
	so, _ := r.Get(0).([]stackConfig)
	return so, r.Error(1)
}

type fakeTemplateStore struct {
	mock.Mock
}

func (f *fakeTemplateStore) Fetch(name string) (Template, error) {
	r := f.Called(name)
	so, _ := r.Get(0).(Template)
	return so, r.Error(1)
}

func TestFetcherFetch(t *testing.T) {

	cs := &fakeConfigStore{}
	ts := &fakeTemplateStore{}

	stackName := "Stack"
	templateName := "Template"

	sc := stackConfig{
		Name:         stackName,
		TemplateName: templateName,
		Parameters: map[string]interface{}{
			"ExistsInTemplate": "abc123",
			"NotInTemplate":    "xyz890",
		},
	}

	tmpl := template{
		parameters: []string{"ExistsInTemplate"},
		body:       "iamthetemplate",
	}

	cs.On("Fetch", stackName).Once().Return([]stackConfig{sc}, nil)
	ts.On("Fetch", templateName).Once().Return(&tmpl, nil)

	r := &paramsResolver{}
	f := newFetcher(cs, ts, r)

	s, err := f.Fetch(stackName)

	// ensure we've got the template body and the correct set of params
	expected := []client.Stack{
		&stack{
			name:         stackName,
			templateBody: tmpl.body,
			rawParameters: map[string]interface{}{
				"ExistsInTemplate": "abc123",
			},
			resolver: r,
		},
	}

	assert.Nil(t, err)
	assert.EqualValues(t, expected, s)
}
