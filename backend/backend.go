package backend

import (
	"os"
	"path"

	"github.com/eyeamera/stacker-cli/stacker"
)

type backend struct {
	f *fetcher
}

func New(dir string) *backend {

	confDir := path.Join(dir, "environments")
	if !pathExists(confDir) {
		confDir = path.Join(dir, "regions")
	}

	cs := newConfigStore(confDir)
	ts := newTemplateStore(path.Join(dir, "templates"))

	r := NewParamsResolver()
	r.Add("Stack", ResolveStackOutput)
	r.Add("File", ResolveFile)

	f := newFetcher(cs, ts, r)

	return &backend{f: f}
}

func (b *backend) FetchAll() ([]stacker.Stack, error) {
	return b.f.FetchAll()
}

func (b *backend) Fetch(name string) ([]stacker.Stack, error) {
	return b.f.Fetch(name)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
