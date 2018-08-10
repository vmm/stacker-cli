package backend

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/awslabs/goformation"
	"github.com/awslabs/goformation/cloudformation"
)

var (
	templateExtensions = []string{".yaml", ".yml", ".json"}
)

type Template interface {
	Body() string
	Parameters() []string
}

type template struct {
	body       string
	parameters []string // List of parameter names
}

func (t *template) Body() string         { return t.body }
func (t *template) Parameters() []string { return t.parameters }

type TemplateStore interface {
	Fetch(name string) (Template, error)
}

type templateStore struct {
	path string
	d    map[string]*template
}

func newTemplateStore(path string) *templateStore {
	return &templateStore{
		path: path,
		d:    make(map[string]*template),
	}
}

func (ts *templateStore) Fetch(name string) (Template, error) {
	if t, ok := ts.d[name]; ok {
		return t, nil
	}

	// check known extensions for template file
	for _, ext := range templateExtensions {
		p := path.Join(ts.path, name+ext)

		if _, err := os.Stat(p); err != nil {
			continue
		}

		t, err := parseTemplate(p)
		if err != nil {
			return nil, err
		}

		ts.d[name] = t

		return t, nil
	}

	return nil, fmt.Errorf("unable to locate template %s", name)
}

func parseTemplate(path string) (*template, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cft *cloudformation.Template
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		cft, err = goformation.ParseYAML(raw)
	} else {
		cft, err = goformation.ParseJSON(raw)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid template %s: %s", path, err)
	}

	p := make([]string, 0)
	for k, _ := range cft.Parameters {
		p = append(p, k)
	}

	return &template{
		body:       string(raw),
		parameters: p,
	}, nil
}
