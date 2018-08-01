package client

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
)

// CloudformationClient provides access to neccessary apis for maniupating stacks
type CloudformationClient interface {
	CreateChangeSet(*cf.CreateChangeSetInput) (*cf.CreateChangeSetOutput, error)
	DeleteStack(*cf.DeleteStackInput) (*cf.DeleteStackOutput, error)
	DescribeChangeSet(*cf.DescribeChangeSetInput) (*cf.DescribeChangeSetOutput, error)
	DescribeStackResources(input *cf.DescribeStackResourcesInput) (*cf.DescribeStackResourcesOutput, error)
	DescribeStacks(*cf.DescribeStacksInput) (*cf.DescribeStacksOutput, error)
	DescribeStacksRequest(*cf.DescribeStacksInput) (*request.Request, *cf.DescribeStacksOutput)
	DescribeStackEvents(*cf.DescribeStackEventsInput) (*cf.DescribeStackEventsOutput, error)
	ExecuteChangeSet(*cf.ExecuteChangeSetInput) (*cf.ExecuteChangeSetOutput, error)
	GetTemplate(*cf.GetTemplateInput) (*cf.GetTemplateOutput, error)
	ListChangeSets(input *cf.ListChangeSetsInput) (*cf.ListChangeSetsOutput, error)
	ListStacksPages(input *cf.ListStacksInput, fn func(*cf.ListStacksOutput, bool) bool) error
	WaitUntilChangeSetCreateCompleteWithContext(ctx aws.Context, input *cf.DescribeChangeSetInput, opts ...request.WaiterOption) error
}

// NewCloudformationClient creates a new CloudformationClient given a region
func NewCloudformationClient(region string) CloudformationClient {
	s, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil
	}
	return cf.New(s)
}
