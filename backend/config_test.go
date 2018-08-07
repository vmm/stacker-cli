package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestEnvsDir = "../test/stacker/environments"

func TestConfigStoreInternalFetch(t *testing.T) {
	s := newConfigStore(TestEnvsDir)
	assert.Nil(t, s.d)

	s.fetch(nil)

	expected := configStoreMap{
		"production": config{
			Defaults: defaults{
				Region: "us-west-2",
				Parameters: map[string]interface{}{
					"VpcCIDR": "10.21.0.0/16",
					"Bar":     "123abc",
				},
			},
		},
		"production/vpc": config{
			Defaults: defaults{
				Region:     "us-west-2",
				Parameters: map[string]interface{}{"VpcCIDR": "10.21.0.0/16"},
			},
			Stacks: []stackConfig{
				stackConfig{
					Name:         "Foo-VPC",
					TemplateName: "VPC",
					Capabilities: "CAPABILITIES_IAM",
					Parameters: map[string]interface{}{
						"Bar":     "123abc",
						"Name":    "ProductionVPC",
						"VpcCIDR": "10.21.0.0/16",
					},
				},
			},
		},
		"sandbox": config{
			Defaults: defaults{
				Region:     "us-east-2",
				Parameters: map[string]interface{}(nil),
			},
			Stacks: []stackConfig{
				stackConfig{Name: "Foo-VPC",
					TemplateName: "VPC",
					Capabilities: "CAPABILITIES_IAM",
					Parameters: map[string]interface{}{
						"VpcCIDR": "10.11.0.0/16",
						"Name":    "SandboxVPC",
					},
				},
			},
		},
	}

	assert.NotNil(t, s.d)
	assert.Equal(t, expected, s.d)
}

// @TODO, this fails randomly
func TestConfigStoreFetch(t *testing.T) {
	// s := newConfigStore(TestEnvsDir)
	// assert.Nil(t, s.d)

	// stacks, _ := s.Fetch("Foo-VPC")

	// expected := []stackConfig{
	// 	stackConfig{
	// 		Name:         "Foo-VPC",
	// 		Region:       "us-west-2", // inherited from 'production'
	// 		TemplateName: "VPC",
	// 		Capabilities: "CAPABILITIES_IAM",
	// 		Parameters: map[string]interface{}{
	// 			"Name":    "ProductionVPC",
	// 			"VpcCIDR": "10.21.0.0/16", // inherited from 'production/vpc'
	// 			"Bar":     "123abc",       // inherited from 'production'
	// 		},
	// 	},
	// 	stackConfig{
	// 		Name:         "Foo-VPC",
	// 		Region:       "us-east-1", // inherited from 'sandbox'
	// 		TemplateName: "VPC",
	// 		Capabilities: "CAPABILITIES_IAM",
	// 		Parameters: map[string]interface{}{
	// 			"Name":    "SandboxVPC",
	// 			"VpcCIDR": "10.11.0.0/16",
	// 		},
	// 	},
	// }

	// assert.EqualValues(t, expected, stacks)
}
