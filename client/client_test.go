package client

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fakeStack struct {
	name         string
	templateBody string
	params       StackParams
	capabilities []string
}

func (s *fakeStack) Name() string                 { return s.name }
func (s *fakeStack) Region() string               { return "" }
func (s *fakeStack) TemplateBody() string         { return s.templateBody }
func (s *fakeStack) Params() (StackParams, error) { return s.params, nil }
func (s *fakeStack) Capabilities() []string       { return s.capabilities }

type fakeStackParam struct {
	key         string
	value       string
	usePrevious bool
}

func (s *fakeStackParam) Key() string       { return s.key }
func (s *fakeStackParam) Value() string     { return s.value }
func (s *fakeStackParam) UsePrevious() bool { return s.usePrevious }

type mockCloudformation struct {
	CloudformationClient
	mock.Mock
}

func (c *mockCloudformation) DescribeStacks(si *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	r := c.Called(si)
	so, _ := r.Get(0).(*cloudformation.DescribeStacksOutput)
	return so, r.Error(1)
}

func (c *mockCloudformation) GetTemplate(si *cloudformation.GetTemplateInput) (*cloudformation.GetTemplateOutput, error) {
	r := c.Called(si)
	so, _ := r.Get(0).(*cloudformation.GetTemplateOutput)
	return so, r.Error(1)
}

func (c *mockCloudformation) ListChangeSets(si *cloudformation.ListChangeSetsInput) (*cloudformation.ListChangeSetsOutput, error) {
	r := c.Called(si)
	so, _ := r.Get(0).(*cloudformation.ListChangeSetsOutput)
	return so, r.Error(1)
}

func (c *mockCloudformation) DescribeChangeSet(si *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
	r := c.Called(si)
	so, _ := r.Get(0).(*cloudformation.DescribeChangeSetOutput)
	return so, r.Error(1)
}

func (c *mockCloudformation) DescribeStackResources(si *cloudformation.DescribeStackResourcesInput) (*cloudformation.DescribeStackResourcesOutput, error) {
	r := c.Called(si)
	so, _ := r.Get(0).(*cloudformation.DescribeStackResourcesOutput)
	return so, r.Error(1)
}

func (c *mockCloudformation) CreateChangeSet(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	r := c.Called(input)
	so, _ := r.Get(0).(*cloudformation.CreateChangeSetOutput)
	return so, r.Error(1)
}

func (c *mockCloudformation) DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	r := c.Called(input)
	so, _ := r.Get(0).(*cloudformation.DeleteStackOutput)
	return so, r.Error(1)
}

func TestGet(t *testing.T) {
	var (
		cf          = &mockCloudformation{}
		c           = New(cf)
		now         = time.Now()
		stackName   = "Foo-Stack"
		stackStatus = "CREATE_COMPLETE"
	)

	scenarios := []struct {
		response *cloudformation.DescribeStacksOutput
		err      error

		expected *StackInfo
		hasError bool
	}{
		{
			&cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackId:         aws.String("foo-stack-id"),
						StackName:       aws.String(stackName),
						StackStatus:     aws.String(stackStatus),
						CreationTime:    aws.Time(now),
						LastUpdatedTime: nil,
						Parameters: []*cloudformation.Parameter{
							{ParameterKey: aws.String("key1"), ParameterValue: aws.String("value1")},
							{ParameterKey: aws.String("key2"), ParameterValue: aws.String("value2")},
						},
						Outputs: []*cloudformation.Output{
							{OutputKey: aws.String("outkey1"), OutputValue: aws.String("outvalue1")},
							{OutputKey: aws.String("outkey2"), OutputValue: aws.String("outvalue2")},
						},
					},
				},
			},
			nil,
			&StackInfo{
				ID:              "foo-stack-id",
				Name:            stackName,
				Status:          stackStatus,
				CreationTime:    now,
				LastUpdatedTime: now,
				Params: StackParamInfos{
					{Key: "key1", Value: "value1"},
					{Key: "key2", Value: "value2"},
				},
				Outputs: StackOutputInfos{
					{Key: "outkey1", Value: "outvalue1"},
					{Key: "outkey2", Value: "outvalue2"},
				},
			},
			false,
		},

		{
			&cloudformation.DescribeStacksOutput{},
			errors.New("some error"),
			nil,
			true,
		},

		// Stack not found case:
		// don't return an error, just a nil StackInfo since
		// we know its just a not-found error
		{
			&cloudformation.DescribeStacksOutput{},
			awserr.NewRequestFailure(
				awserr.New("ValidationError", "Oh noes", errors.New("boom")), 400, "reqid",
			),
			nil,
			false,
		},
	}

	for _, s := range scenarios {
		cf.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}).
			Once().
			Return(s.response, s.err)

		si, err := c.Get(stackName)
		assert.Equal(t, s.expected, si)

		if s.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	cf := &mockCloudformation{}
	c := New(cf)

	stackName := "Foo-Stack"

	scenarios := []struct {
		response *cloudformation.GetTemplateOutput
		err      error
		expected string
		hasError bool
	}{
		{
			&cloudformation.GetTemplateOutput{
				TemplateBody: aws.String("thetemplate."),
			},
			nil,
			"thetemplate.",
			false,
		},

		{
			&cloudformation.GetTemplateOutput{},
			errors.New("boom"),
			"",
			true,
		},
	}

	for _, s := range scenarios {
		cf.On("GetTemplate", &cloudformation.GetTemplateInput{
			StackName:     aws.String(stackName),
			TemplateStage: aws.String("Processed"),
		}).Once().Return(s.response, s.err)

		si, err := c.GetTemplate(stackName)
		assert.Equal(t, s.expected, si)
		if s.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestGetChangeSets(t *testing.T) {
	var (
		cf        = &mockCloudformation{}
		c         = New(cf)
		now       = time.Now()
		stackName = "Foo-Stack"
	)

	scenarios := []struct {
		response *cloudformation.ListChangeSetsOutput
		err      error

		expected PendingChangeSets
		hasError bool
	}{
		{
			&cloudformation.ListChangeSetsOutput{
				Summaries: []*cloudformation.ChangeSetSummary{
					{
						ChangeSetId:   aws.String("cs-1"),
						ChangeSetName: aws.String("cs-a"),
						CreationTime:  aws.Time(now),
						StackId:       aws.String("stack-a"),
						StackName:     aws.String(stackName),
						Status:        aws.String("CREATE_FAILED"),
						StatusReason:  aws.String("something went wrong"),
					},
					{
						ChangeSetId:   aws.String("cs-2"),
						ChangeSetName: aws.String("cs-b"),
						CreationTime:  aws.Time(now),
						StackId:       aws.String("stack-b"),
						StackName:     aws.String(stackName),
						Status:        aws.String("CREATE_COMPLETE"),
					},
				},
			},
			nil,
			[]PendingChangeSet{
				{
					ID:           "cs-1",
					Name:         "cs-a",
					CreationTime: now,
					StackID:      "stack-a",
					StackName:    stackName,
					Status:       "CREATE_FAILED",
					StatusReason: "something went wrong",
				},
				{
					ID:           "cs-2",
					Name:         "cs-b",
					CreationTime: now,
					StackID:      "stack-b",
					StackName:    stackName,
					Status:       "CREATE_COMPLETE",
				},
			},
			false,
		},

		{
			&cloudformation.ListChangeSetsOutput{},
			errors.New("boom"),
			nil,
			true,
		},
	}

	for _, s := range scenarios {
		cf.On("ListChangeSets", &cloudformation.ListChangeSetsInput{StackName: aws.String(stackName)}).
			Once().
			Return(s.response, s.err)

		si, err := c.GetChangeSets(stackName)
		assert.Equal(t, s.expected, si)

		if s.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestGetChangeSet(t *testing.T) {
	var (
		cf        = &mockCloudformation{}
		c         = New(cf)
		now       = time.Now()
		stackName = "Foo-Stack"
		changeSet = "cs-12345678"
	)

	scenarios := []struct {
		response *cloudformation.DescribeChangeSetOutput
		err      error

		expected *ChangeSetInfo
		hasError bool
	}{
		{
			&cloudformation.DescribeChangeSetOutput{
				ChangeSetId:     aws.String("id"),
				ChangeSetName:   aws.String(changeSet),
				StackId:         aws.String("stackid"),
				StackName:       aws.String(stackName),
				ExecutionStatus: aws.String("AVAILABLE"),
				Status:          aws.String("CREATE_COMPELETE"),
				StatusReason:    aws.String("it worked"),
				CreationTime:    aws.Time(now),
			},
			nil,
			&ChangeSetInfo{
				ID:              "id",
				Name:            changeSet,
				Status:          "CREATE_COMPELETE",
				StatusReason:    "it worked",
				ExecutionStatus: "AVAILABLE",
				CreationTime:    now,
				StackID:         "stackid",
				StackName:       stackName,
				Changes:         ResourceChanges{},
				Params:          StackParamInfos{},
			},
			false,
		},

		{
			&cloudformation.DescribeChangeSetOutput{},
			errors.New("boom"),
			nil,
			true,
		},
	}

	for _, s := range scenarios {
		cf.On("DescribeChangeSet", &cloudformation.DescribeChangeSetInput{
			StackName:     aws.String(stackName),
			ChangeSetName: aws.String(changeSet)},
		).Once().Return(s.response, s.err)

		si, err := c.GetChangeSet(stackName, changeSet)
		assert.Equal(t, s.expected, si)

		if s.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestGetResources(t *testing.T) {
	var (
		cf        = &mockCloudformation{}
		c         = New(cf)
		now       = time.Now()
		stackName = "Foo-Stack"
	)

	scenarios := []struct {
		response *cloudformation.DescribeStackResourcesOutput
		err      error

		expected ResourceInfos
		hasError bool
	}{

		{
			&cloudformation.DescribeStackResourcesOutput{
				StackResources: []*cloudformation.StackResource{
					{
						PhysicalResourceId: aws.String("id1"),
						LogicalResourceId:  aws.String("name"),
						ResourceStatus:     aws.String("status"),
						ResourceType:       aws.String("AWS::FOO:Resource"),
						Timestamp:          aws.Time(now),
					},
					{
						PhysicalResourceId: aws.String("id2"),
						LogicalResourceId:  aws.String("name"),
						ResourceStatus:     aws.String("status"),
						ResourceType:       aws.String("AWS::FOO:Resource"),
						Timestamp:          aws.Time(now),
					},
				},
			},
			nil,
			[]ResourceInfo{
				{
					ID:          "id1",
					Name:        "name",
					Status:      "status",
					Type:        "AWS::FOO:Resource",
					UpdatedTime: now,
				},
				{
					ID:          "id2",
					Name:        "name",
					Status:      "status",
					Type:        "AWS::FOO:Resource",
					UpdatedTime: now,
				},
			},
			false,
		},

		{
			&cloudformation.DescribeStackResourcesOutput{},
			errors.New("boom"),
			nil,
			true,
		},
	}

	for _, s := range scenarios {
		cf.On("DescribeStackResources", &cloudformation.DescribeStackResourcesInput{
			StackName: aws.String(stackName),
		}).Once().Return(s.response, s.err)

		si, err := c.GetResources(stackName)
		assert.Equal(t, s.expected, si)

		if s.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestCreateChangeSet(t *testing.T) {
	var (
		cf        = &mockCloudformation{}
		c         = New(cf)
		changeSet = "cs-12345678"
		stackName = "Foo-Stack"
		now       = time.Now()
	)

	scenarios := []struct {
		stack       Stack
		createErr   error
		getResponse *cloudformation.DescribeChangeSetOutput
		getErr      error

		expected *ChangeSetInfo
		hasError bool
	}{
		{
			&fakeStack{
				name:         stackName,
				templateBody: "the-template",
				params: StackParams{
					&fakeStackParam{key: "Name", value: "FooVPC"},
					&fakeStackParam{key: "VpcCIDR", value: "10.111.0.0/16"},
				},
			},
			nil,
			&cloudformation.DescribeChangeSetOutput{
				ChangeSetId:     aws.String("id"),
				ChangeSetName:   aws.String(changeSet),
				StackId:         aws.String("stackid"),
				StackName:       aws.String(stackName),
				ExecutionStatus: aws.String("AVAILABLE"),
				Status:          aws.String("CREATE_COMPELETE"),
				StatusReason:    aws.String("it worked"),
				CreationTime:    aws.Time(now),
			},
			nil,
			&ChangeSetInfo{
				ID:              "id",
				Name:            changeSet,
				Status:          "CREATE_COMPELETE",
				StatusReason:    "it worked",
				ExecutionStatus: "AVAILABLE",
				CreationTime:    now,
				StackID:         "stackid",
				StackName:       stackName,
				Changes:         ResourceChanges{},
				Params:          StackParamInfos{},
			},
			false,
		},

		{
			&fakeStack{
				name:         stackName,
				templateBody: "the-template",
				params: StackParams{
					&fakeStackParam{key: "Name", value: "FooVPC"},
					&fakeStackParam{key: "VpcCIDR", value: "10.111.0.0/16"},
				},
			},
			errors.New("Boom"),
			&cloudformation.DescribeChangeSetOutput{},
			nil,
			nil,
			true,
		},

		{
			&fakeStack{
				name:         stackName,
				templateBody: "the-template",
				params: StackParams{
					&fakeStackParam{key: "Name", value: "FooVPC"},
					&fakeStackParam{key: "VpcCIDR", value: "10.111.0.0/16"},
				},
			},
			nil,
			&cloudformation.DescribeChangeSetOutput{},
			errors.New("Boom"),
			nil,
			true,
		},
	}

	for _, s := range scenarios {
		p, _ := s.stack.Params()

		cf.On("CreateChangeSet", &cloudformation.CreateChangeSetInput{
			ChangeSetName: aws.String(changeSet),
			ChangeSetType: aws.String(cloudformation.ChangeSetTypeCreate),
			StackName:     aws.String(s.stack.Name()),
			TemplateBody:  aws.String(s.stack.TemplateBody()),
			Parameters:    cfParams(p),
		}).Once().Return(nil, s.createErr)

		if s.createErr == nil {
			cf.On("DescribeChangeSet", &cloudformation.DescribeChangeSetInput{
				StackName:     aws.String(s.stack.Name()),
				ChangeSetName: aws.String(changeSet),
			}).Once().Return(s.getResponse, s.getErr)
		}

		si, err := c.createChangeSet(cloudformation.ChangeSetTypeCreate, changeSet, s.stack)
		assert.Equal(t, s.expected, si)

		if s.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestDelete(t *testing.T) {
	var (
		cf        = &mockCloudformation{}
		c         = New(cf)
		stackName = "Foo-Stack"
	)

	scenarios := []struct {
		err error
	}{
		{nil},
		{errors.New("boom")},
	}

	for _, s := range scenarios {
		cf.On("DeleteStack", &cloudformation.DeleteStackInput{
			StackName: aws.String(stackName),
		}).Once().Return(&cloudformation.DeleteStackOutput{}, s.err)

		err := c.Delete(stackName)

		if s.err != nil {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}
