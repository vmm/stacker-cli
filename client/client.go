package client

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"
)

// StackParam represents a stack parameter
type StackParam interface {
	Key() string
	Value() string
	UsePrevious() bool
}

// StackParams represents an set of StackParam
type StackParams []StackParam

// Stack interface for creating and updating stacks
type Stack interface {
	Name() string
	Region() string
	Params() (StackParams, error)
	TemplateBody() string
	Capabilities() []string
}

// Sortable list of Stacks
type StackList []Stack

func (s StackList) Len() int {
	return len(s)
}

func (s StackList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s StackList) Less(i, j int) bool {
	return s[i].Name() < s[j].Name()
}

// Sortable list of StackInfos
type StackInfoList []*StackInfo

func (s StackInfoList) Len() int {
	return len(s)
}

func (s StackInfoList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s StackInfoList) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// Client performs Cloudformation actions with the native Stack interface
type Client struct {
	cf CloudformationClient
}

// New returns a new Client given a CloudformationClient
func New(cf CloudformationClient) *Client {
	return &Client{cf: cf}
}

func (c *Client) ListStacks() ([]*StackInfo, error) {
	stackInfos := []*StackInfo{}

	err := c.cf.ListStacksPages(&cf.ListStacksInput{
		StackStatusFilter: []*string{
			aws.String(cf.StackStatusCreateComplete),
			aws.String(cf.StackStatusRollbackFailed),
			aws.String(cf.StackStatusRollbackComplete),
			aws.String(cf.StackStatusDeleteFailed),
			aws.String(cf.StackStatusUpdateComplete),
			aws.String(cf.StackStatusUpdateRollbackFailed),
			aws.String(cf.StackStatusUpdateRollbackComplete),
			aws.String(cf.StackStatusCreateInProgress),
			aws.String(cf.StackStatusRollbackInProgress),
			aws.String(cf.StackStatusUpdateInProgress),
			aws.String(cf.StackStatusUpdateCompleteCleanupInProgress),
			aws.String(cf.StackStatusUpdateRollbackInProgress),
			aws.String(cf.StackStatusUpdateRollbackCompleteCleanupInProgress),
		},
	}, func(output *cf.ListStacksOutput, last bool) bool {
		for _, s := range output.StackSummaries {
			si := &StackInfo{
				ID:              deref(s.StackId),
				Name:            *s.StackName,
				Status:          *s.StackStatus,
				CreationTime:    *s.CreationTime,
				LastUpdatedTime: *s.CreationTime,
				Params:          StackParamInfos{},
				Outputs:         StackOutputInfos{},
			}

			if s.LastUpdatedTime != nil {
				si.LastUpdatedTime = *s.LastUpdatedTime
			}

			stackInfos = append(stackInfos, si)
		}

		return true
	})

	if err != nil {
		rerr, ok := err.(awserr.RequestFailure)
		if !ok || (rerr.StatusCode() != 400 && rerr.Code() != "ValidationError") {
			return nil, errors.Wrap(err, "unable to fetch stack")
		}
		return nil, nil
	}

	return stackInfos, nil
}

// Exists checks the existence of a stack provided its name
func (c *Client) Exists(stackName string) (bool, error) {
	s, err := c.Get(stackName)
	return s != nil, err
}

// Get retrieves information about a stack
func (c *Client) Get(stackName string) (*StackInfo, error) {
	output, err := c.cf.DescribeStacks(&cf.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		rerr, ok := err.(awserr.RequestFailure)
		if !ok || (rerr.StatusCode() != 400 && rerr.Code() != "ValidationError") {
			return nil, errors.Wrap(err, "unable to fetch stack")
		}
		return nil, nil
	}

	for _, s := range output.Stacks {
		if *s.StackName == stackName {
			return newStackInfo(s), nil
		}
	}

	return nil, nil
}

// GetTemplate retrieves a stack's underlying template
func (c *Client) GetTemplate(stackName string) (string, error) {
	input := &cf.GetTemplateInput{
		StackName:     aws.String(stackName),
		TemplateStage: aws.String(cf.TemplateStageProcessed),
	}
	output, err := c.cf.GetTemplate(input)
	if err != nil {
		return "", errors.Wrap(err, "unable to fetch template")
	}

	if output.TemplateBody != nil {
		return *output.TemplateBody, nil
	}
	return "", nil
}

// GetChangeSets returns the pending, uncommitted changesets for a stack
func (c *Client) GetChangeSets(stackName string) (PendingChangeSets, error) {
	output, err := c.cf.ListChangeSets(&cf.ListChangeSetsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch changesets")
	}

	return newPendingChangeSets(output.Summaries), nil
}

// GetChangeSet returns information about a pending changeset
func (c *Client) GetChangeSet(stackName string, changeSetName string) (*ChangeSetInfo, error) {
	output, err := c.cf.DescribeChangeSet(&cf.DescribeChangeSetInput{
		StackName:     aws.String(stackName),
		ChangeSetName: aws.String(changeSetName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch changeset")
	}

	return newChangeSetInfo(output), nil
}

// GetResources fetches a Stack's resource information
func (c *Client) GetResources(stackName string) (ResourceInfos, error) {
	output, err := c.cf.DescribeStackResources(&cf.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch resources")
	}

	return newResourceInfos(output.StackResources), nil
}

// GetEvents returns the latest events for a stack
func (c *Client) GetEvents(stackName string) (StackEvents, error) {
	output, err := c.cf.DescribeStackEvents(&cf.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch resources")
	}

	return newStackEvents(output.StackEvents), nil
}

// Create creates a changeset for creating a new stack
func (c *Client) Create(s Stack) (*ChangeSetInfo, error) {
	changeSetName, err := changeSetName()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create changeset name")
	}

	return c.createChangeSet(cf.ChangeSetTypeCreate, changeSetName, s)
}

// Update creates a changeset for updating an existing stack
func (c *Client) Update(s Stack) (*ChangeSetInfo, error) {
	changeSetName, err := changeSetName()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create changeset name")
	}

	return c.createChangeSet(cf.ChangeSetTypeUpdate, changeSetName, s)
}

// Commit commits a pending change set
func (c *Client) Commit(stackName string, changeSetName string) error {
	_, err := c.cf.ExecuteChangeSet(&cf.ExecuteChangeSetInput{
		StackName:     aws.String(stackName),
		ChangeSetName: aws.String(changeSetName),
	})
	return errors.Wrap(err, "unable to commit changeset")
}

// Delete deletes a stack
func (c *Client) Delete(name string) error {
	_, err := c.cf.DeleteStack(&cf.DeleteStackInput{
		StackName: aws.String(name),
	})
	return errors.Wrap(err, "unable to delete stack")
}

// WaitForChangeSetComplete blocks until a change set has been created
func (c *Client) WaitForChangeSetComplete(stackName string, changeSetName string) error {
	return c.cf.WaitUntilChangeSetCreateCompleteWithContext(
		aws.BackgroundContext(),
		&cf.DescribeChangeSetInput{
			StackName:     aws.String(stackName),
			ChangeSetName: aws.String(changeSetName),
		},
		request.WithWaiterDelay(request.ConstantWaiterDelay(5*time.Second)),
	)
}

// WaitForStackComplete blocks until a stack has finished updating
func (c *Client) WaitForStackComplete(stackName string) error {
	ctx := aws.BackgroundContext()
	input := &cf.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	w := request.Waiter{
		Name:        "WaitUntilStackComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(5 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "CREATE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "CREATE_FAILED",
			},
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "ROLLBACK_COMPLETE",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		// Logger: cf.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *cf.DescribeStacksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := c.cf.DescribeStacksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	return w.WaitWithContext(ctx)
}

func (c *Client) createChangeSet(typ string, changeSetName string, s Stack) (*ChangeSetInfo, error) {
	if !(typ == cf.ChangeSetTypeCreate || typ == cf.ChangeSetTypeUpdate) {
		return nil, fmt.Errorf("unknown changeset type \"%s\"", typ)
	}

	params, err := s.Params()
	if err != nil {
		return nil, err
	}

	cs := &cf.CreateChangeSetInput{
		ChangeSetName: aws.String(changeSetName),
		ChangeSetType: aws.String(typ),
		StackName:     aws.String(s.Name()),
		TemplateBody:  aws.String(s.TemplateBody()),
		Parameters:    cfParams(params),
	}

	if len(s.Capabilities()) > 0 {
		caps := make([]*string, len(s.Capabilities()))
		for i, c := range s.Capabilities() {
			caps[i] = aws.String(c)
		}
		cs.Capabilities = caps
	}

	if _, err := c.cf.CreateChangeSet(cs); err != nil {
		return nil, errors.Wrap(err, "unable to create changeset")
	}

	return c.GetChangeSet(s.Name(), changeSetName)
}

func cfParams(sp StackParams) []*cf.Parameter {
	params := make([]*cf.Parameter, len(sp))
	for i, p := range sp {
		param := &cf.Parameter{ParameterKey: aws.String(p.Key())}
		if p.UsePrevious() {
			param.UsePreviousValue = aws.Bool(true)
		} else {
			param.ParameterValue = aws.String(p.Value())
		}
		params[i] = param
	}
	return params
}

// changeSetName provides a random name
func changeSetName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// "Member must satisfy regular expression pattern: [a-zA-Z][-a-zA-Z0-9]*"
	name := fmt.Sprintf("cs-%x", md5.Sum(b))
	return name[:11], nil
}

// NotifyUntilComplete blocks until a stack update is complete, periodically
// calling the provided callback
func (c *Client) NotifyUntilComplete(name string, f func(s *StackInfo)) error {
	exists, err := c.Exists(name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("stack %s does not exist", name)
	}

	for {
		stack, err := c.Get(name)
		if err != nil {
			return err
		}

		// @TODO stack could not exist here...
		if stack == nil {
			return fmt.Errorf("stack %s does not exist", name)
		}

		f(stack)

		switch stack.Status {
		case cf.StackStatusCreateFailed,
			cf.StackStatusCreateComplete,
			cf.StackStatusRollbackFailed,
			cf.StackStatusRollbackComplete,
			cf.StackStatusDeleteFailed,
			cf.StackStatusDeleteComplete,
			cf.StackStatusUpdateComplete,
			cf.StackStatusUpdateRollbackFailed,
			cf.StackStatusUpdateRollbackComplete:
			return nil
			// case cf.StackStatusCreateInProgress,
			// 	cf.StackStatusRollbackInProgress,
			// 	cf.StackStatusDeleteInProgress,
			// 	cf.StackStatusUpdateInProgress,
			// 	cf.StackStatusUpdateCompleteCleanupInProgress,
			// 	cf.StackStatusUpdateRollbackInProgress,
			// 	cf.StackStatusUpdateRollbackCompleteCleanupInProgress,
			// 	cf.StackStatusReviewInProgress:
			// Keep looping..
		}

		time.Sleep(5 * time.Second)
	}
	return nil
}
