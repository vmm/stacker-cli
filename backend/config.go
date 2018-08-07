package backend

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

// config represents a config file from disk
type config struct {
	Defaults defaults
	Stacks   []stackConfig
}

type defaults struct {
	Region     string
	Parameters map[string]interface{}
}

type stackConfig struct {
	Name         string
	Region       string
	TemplateName string `yaml:"template_name"`
	Capabilities string
	Parameters   map[string]interface{}
}

type ConfigStore interface {
	FetchAll() ([]stackConfig, error)
	Fetch(name string) ([]stackConfig, error)
}

// storeMap stores a map of the relative config path with extensions removed
// to the config struct
type configStoreMap map[string]config

// store provides a way for retrieving configuration files
// and stack configs from the filesystem
type configStore struct {
	path string
	d    configStoreMap
}

func newConfigStore(path string) *configStore {
	return &configStore{path: path}
}

func (s *configStore) FetchAll() ([]stackConfig, error) {
	return s.fetch(nil)
}

func (s *configStore) Fetch(name string) ([]stackConfig, error) {
	return s.fetch(&name)
}

// Fetch returns a slice of stacks that match the provided stack name.
// Each stackConfig will contain all of the inherited parameters from
// the parent config file(s)
func (s *configStore) fetch(name *string) ([]stackConfig, error) {
	// Load the config files from disk
	if err := s.load(); err != nil {
		return nil, err
	}

	scs := make([]stackConfig, 0)

	for path, config := range s.d {
		for _, stack := range config.Stacks {
			if name == nil || stack.Name == *name {
				scs = append(scs, s.resolveStack(path, stack))
			}
		}
	}

	return scs, nil
}

// resolveStack takes a path and a stackConfig and merges default parameters
// into a returned stackConfig
func (s *configStore) resolveStack(path string, st stackConfig) stackConfig {
	stack := st

	// Apply default parameters from parent config paths
	for path != "." {
		c, ok := s.d[path]
		if !ok {
			path = filepath.Dir(path)
			continue
		}

		if stack.Region == "" && c.Defaults.Region != "" {
			stack.Region = c.Defaults.Region
		}

		if stack.Parameters == nil {
			stack.Parameters = map[string]interface{}{}
		}

		if stack.TemplateName == "" {
			stack.TemplateName = stack.Name
		}

		for k, v := range c.Defaults.Parameters {
			if _, ok := stack.Parameters[k]; ok {
				continue
			}
			stack.Parameters[k] = v
		}

		// Lop a segment off the path and continue...
		path = filepath.Dir(path)
	}

	return stack
}

// loads all the config files from disk and stores them into a mapping
// of config path to configuration struct.
//
// for example:
//   production => config
//   production/vpc => config (params here will inherit from production)
//   sandbox => config
//   sandbox/vpc => config (params here will inherit from sandbox)
func (s *configStore) load() error {
	if s.d != nil {
		return nil
	}

	s.d = make(configStoreMap)

	// Walk all the files in the config path
	return filepath.Walk(s.path, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !(ext == ".yml" || ext == ".yaml") {
			return nil
		}

		key := path[len(s.path)+1 : len(path)-len(ext)]

		r, err := os.Open(path)
		if err != nil {
			return err
		}

		c, err := readConfig(r)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error parsing file at %s", path))
		}

		s.d[key] = c

		return nil
	})
}

// readConfig unmarshals data from a reader into a config struct
func readConfig(r io.Reader) (config, error) {
	c := config{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}
