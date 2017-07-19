package backend

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestTemplatesDir = "../test/stacker/templates"

func TestTemplateStoreFetch(t *testing.T) {
	ts := newTemplateStore(TestTemplatesDir)

	template, _ := ts.Fetch("VPCYaml")

	data, _ := ioutil.ReadFile(path.Join(TestTemplatesDir, "VPCYaml.yml"))
	body := string(data)

	assert.EqualValues(t, []string{"Name", "VpcCIDR"}, template.Parameters())
	assert.Equal(t, body, template.Body())
}
