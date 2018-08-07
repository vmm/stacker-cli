package backend

import (
	"os"
	"path"

	"github.com/eyeamera/stacker-cli/stacker"
)

type backend struct {
	f *fetcher
}

var backendPaths = []string{
	"environments",
	"regions",
	"stacks",
}

func New(dir string) *backend {
	var confDir string

	for _, confPath := range backendPaths {
		confDir = path.Join(dir, confPath)
		if pathExists(confDir) {
			break
		}
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
