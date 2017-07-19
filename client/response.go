package client

import (
	"bytes"
	"fmt"
	"time"

	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Formatters
var (
	bold      = color.New(color.Bold).SprintFunc()
	underline = color.New(color.Underline).SprintFunc()
	cyan      = color.New(color.FgCyan).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
)

type StackEvents []StackEvent

func (se StackEvents) String() string {
	var buffer bytes.Buffer

	data := make([][]string, len(se))
	for i, e := range se {
		data[i] = []string{
			underline(bold(e.Resource.Type)),
			cyan(e.Resource.Name),
			// cyan(e.Resource.ID),
			cyan(e.Resource.Status),
			cyan(e.Resource.StatusReason),
			cyan(e.Timestamp),
		}
	}

	table := tablewriter.NewWriter(&buffer)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	// table.SetAutoMergeCells(true)
	table.SetAutoWrapText(false)
	table.AppendBulk(data)
	table.Render()

	return buffer.String()
}

func newStackEvents(events []*cf.StackEvent) StackEvents {
	se := make(StackEvents, len(events))
	for i, e := range events {
		se[i] = newStackEvent(e)
	}
	return se
}

// StackEvent represents a modification event for a Stack
type StackEvent struct {
	ID        string
	StackID   string
	StackName string
	Resource  ResourceInfo
	Timestamp time.Time
}

func newStackEvent(e *cf.StackEvent) StackEvent {
	return StackEvent{
		ID:        deref(e.EventId),
		StackID:   deref(e.StackId),
		StackName: deref(e.StackName),
		Resource:  newResourceInfoFromStackEvent(e),
		Timestamp: *e.Timestamp,
	}
}

type ResourceInfos []ResourceInfo

func newResourceInfos(resources []*cf.StackResource) ResourceInfos {
	ri := make(ResourceInfos, len(resources))
	for i, r := range resources {
		ri[i] = newResourceInfo(r)
	}
	return ri
}

func (ri ResourceInfos) String() string {
	var buffer bytes.Buffer

	data := make([][]string, len(ri))
	for i, r := range ri {
		data[i] = []string{
			underline(bold(r.Type)),
			cyan(r.Name),
			cyan(r.Status),
			cyan(r.ID),
			cyan(r.UpdatedTime),
		}
	}

	table := tablewriter.NewWriter(&buffer)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	table.SetAutoMergeCells(true)
	table.SetAutoWrapText(false)
	table.AppendBulk(data)
	table.Render()

	return buffer.String()
}

// ResourceInfo represents a physical AWS resource
type ResourceInfo struct {
	ID           string
	Name         string
	Status       string
	StatusReason string
	Type         string
	UpdatedTime  time.Time
}

func newResourceInfo(s *cf.StackResource) ResourceInfo {
	return ResourceInfo{
		ID:           deref(s.PhysicalResourceId),
		Name:         *s.LogicalResourceId,
		Type:         *s.ResourceType,
		Status:       *s.ResourceStatus,
		StatusReason: deref(s.ResourceStatusReason),
		UpdatedTime:  *s.Timestamp,
	}
}

func newResourceInfoFromStackEvent(e *cf.StackEvent) ResourceInfo {
	return ResourceInfo{
		ID:           deref(e.PhysicalResourceId),
		Name:         deref(e.LogicalResourceId),
		Type:         deref(e.ResourceType),
		Status:       deref(e.ResourceStatus),
		StatusReason: deref(e.ResourceStatusReason),
	}
}

// ResourceChangeDetails is a list of ResourceChangeDetail
type ResourceChangeDetails []ResourceChangeDetail

func newResourceChangeDetails(details []*cf.ResourceChangeDetail) ResourceChangeDetails {
	rcd := make(ResourceChangeDetails, len(details))
	for i, d := range details {
		rcd[i] = newResourceChangeDetail(d)
	}
	return rcd
}

// ResourceChangeDetail describes a ResourceChange
type ResourceChangeDetail struct {
	CausingEntity      string // CuasingEntity
	ChangeSource       string // ChangeSource
	Evaluation         string // Evaluation
	Attribute          string // Target.Attribute
	RequiresRecreation string // Target.RequiresRecreation
}

func newResourceChangeDetail(detail *cf.ResourceChangeDetail) ResourceChangeDetail {
	rcd := ResourceChangeDetail{
		ChangeSource:       *detail.ChangeSource,
		Evaluation:         *detail.Evaluation,
		Attribute:          *detail.Target.Attribute,
		RequiresRecreation: *detail.Target.RequiresRecreation,
	}

	if detail.CausingEntity != nil {
		rcd.CausingEntity = *detail.CausingEntity
	}

	return rcd
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	Action       string
	Name         string // LogicalResourceId (Name provided in stack template)
	ResourceType string
	ResourceID   string
	Replacement  bool
	Details      ResourceChangeDetails
}

func newResourceChange(rc *cf.ResourceChange) ResourceChange {
	c := ResourceChange{
		Action:       *rc.Action,
		Name:         *rc.LogicalResourceId,
		ResourceType: *rc.ResourceType,
		Details:      newResourceChangeDetails(rc.Details),
	}

	if rc.PhysicalResourceId != nil {
		c.ResourceID = *rc.PhysicalResourceId
	}

	if rc.Replacement != nil {
		c.Replacement = *rc.Replacement == "True"
	}

	return c
}

// ResourceChanges is a list of ResourceChange
type ResourceChanges []ResourceChange

func newResourceChanges(changes []*cf.Change) ResourceChanges {
	rc := make(ResourceChanges, len(changes))
	for i, c := range changes {
		rc[i] = newResourceChange(c.ResourceChange)
	}
	return rc
}

// ChangeSetInfo represents a change set to be applied to a stack
type ChangeSetInfo struct {
	ID              string
	Name            string
	Status          string
	StatusReason    string
	ExecutionStatus string
	CreationTime    time.Time
	StackID         string
	StackName       string
	Changes         ResourceChanges
	Params          StackParamInfos
}

func newChangeSetInfo(cso *cf.DescribeChangeSetOutput) *ChangeSetInfo {
	csi := &ChangeSetInfo{
		ID:              deref(cso.ChangeSetId),
		Name:            deref(cso.ChangeSetName),
		StackID:         deref(cso.StackId),
		StackName:       deref(cso.StackName),
		ExecutionStatus: deref(cso.ExecutionStatus),
		Status:          deref(cso.Status),
		StatusReason:    deref(cso.StatusReason),
		Changes:         newResourceChanges(cso.Changes),
		Params:          newStackParamInfos(cso.Parameters),
	}

	if cso.CreationTime != nil {
		csi.CreationTime = *cso.CreationTime
	}

	return csi
}

func (cs *ChangeSetInfo) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("%s: %s\n", underline(bold("Stack")), cyan(cs.StackName)))
	buffer.WriteString(fmt.Sprintf("%s: %s\n", underline(bold("Changeset")), cyan(cs.Name)))

	var status string
	switch cs.Status {
	case cf.ChangeSetStatusFailed:
		status = red(cs.Status)
	default:
		status = cyan(cs.Status)
	}
	buffer.WriteString(fmt.Sprintf("  %s: %s\n", bold("Status"), status))
	if cs.StatusReason != "" {
		buffer.WriteString(fmt.Sprintf("  %s: %s\n", bold("Reason"), red(cs.StatusReason)))
	}

	var executionStatus string
	switch cs.ExecutionStatus {
	case cf.ExecutionStatusUnavailable:
		executionStatus = red(cs.ExecutionStatus)
	default:
		executionStatus = cyan(cs.ExecutionStatus)
	}

	buffer.WriteString(fmt.Sprintf("  %s: %s\n", bold("Execution Status"), executionStatus))
	buffer.WriteString(fmt.Sprintf("  %s: %s\n", bold("Created At"), cyan(cs.CreationTime)))

	if len(cs.Changes) > 0 {
		buffer.WriteString(fmt.Sprintf("  %s\n", bold("Resources")))
	}

	for _, r := range cs.Changes {

		if r.ResourceID == "" {
			buffer.WriteString(fmt.Sprintf("    %s: %s\n", underline(bold(r.ResourceType)), cyan(r.Name)))
		} else {
			buffer.WriteString(fmt.Sprintf("    %s: %s (%s)\n", underline(bold(r.ResourceType)), cyan(r.Name), cyan(r.ResourceID)))
		}

		var action string
		switch r.Action {
		case "Add":
			action = green(r.Action)
		case "Modify":
			action = yellow(r.Action)
		case "Remove":
			action = red(r.Action)
		}

		buffer.WriteString(fmt.Sprintf("      %s: %s\n", bold("Action"), action))

		if r.Replacement {
			buffer.WriteString(fmt.Sprintf("      %s: %s\n", bold("Replacement"), red(r.Replacement)))
		} else {
			buffer.WriteString(fmt.Sprintf("      %s: %s\n", bold("Replacement"), cyan(r.Replacement)))
		}

		// if len(r.Details) == 0 {
		// 	continue
		// }

		// buffer.WriteString(fmt.Sprintf("      %s\n", bold("Details")))

		// for _, d := range r.Details {
		// 	buffer.WriteString(fmt.Sprintf("        %s\n", bold(d.ChangeSource)))
		// 	if d.CausingEntity != "" {
		// 		buffer.WriteString(fmt.Sprintf("          %s: %s\n", bold("Entity"), cyan(d.CausingEntity)))
		// 	}
		// 	buffer.WriteString(fmt.Sprintf("          %s: %s\n", bold("Evaluation"), cyan(d.Evaluation)))
		// 	buffer.WriteString(fmt.Sprintf("          %s: %s\n", bold("Attribute"), cyan(d.Attribute)))
		// 	buffer.WriteString(fmt.Sprintf("          %s: %s\n", bold("RequiresRecreation"), cyan(d.RequiresRecreation)))
		// }
	}

	return buffer.String()
}

// CanCommit returns whether a changeset can be committed
func (cs *ChangeSetInfo) CanCommit() bool {
	return cs.ExecutionStatus == cf.ExecutionStatusAvailable
}

// PendingChangeSets is a list of PendingChangeSet
type PendingChangeSets []PendingChangeSet

func newPendingChangeSets(summaries []*cf.ChangeSetSummary) PendingChangeSets {
	pcs := make(PendingChangeSets, len(summaries))
	for i, s := range summaries {
		pcs[i] = newPendingChangeSet(s)
	}
	return pcs
}

// PendingChangeSet represents a changeset that has not been committed
type PendingChangeSet struct {
	ID              string
	Name            string
	Status          string
	StatusReason    string
	ExecutionStatus string
	CreationTime    time.Time
	StackID         string
	StackName       string
}

func newPendingChangeSet(s *cf.ChangeSetSummary) PendingChangeSet {
	p := PendingChangeSet{
		ID:              deref(s.ChangeSetId),
		Name:            deref(s.ChangeSetName),
		StackID:         deref(s.StackId),
		StackName:       deref(s.StackName),
		ExecutionStatus: deref(s.ExecutionStatus),
		Status:          deref(s.Status),
		StatusReason:    deref(s.StatusReason),
	}

	if s.CreationTime != nil {
		p.CreationTime = *s.CreationTime
	}

	return p
}

// StackParamInfos is a list of StackParamInfo
type StackParamInfos []StackParamInfo

func newStackParamInfos(params []*cf.Parameter) StackParamInfos {
	spi := make(StackParamInfos, len(params))
	for i, p := range params {
		spi[i] = newStackParamInfo(p)
	}
	return spi
}

func (spi StackParamInfos) String() string {
	var buffer bytes.Buffer

	data := make([][]string, len(spi))

	for i, p := range spi {
		data[i] = []string{"", bold(p.Key), cyan(p.Value)}
	}

	table := tablewriter.NewWriter(&buffer)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()

	return buffer.String()
}

// StackParamInfo represents a Stack's parameters
type StackParamInfo struct {
	Key   string
	Value string
}

func newStackParamInfo(param *cf.Parameter) StackParamInfo {
	return StackParamInfo{
		Key:   deref(param.ParameterKey),
		Value: deref(param.ParameterValue),
	}
}

// StackOutputInfos is a list of StackParamInfo
type StackOutputInfos []StackOutputInfo

func newStackOutputInfos(outputs []*cf.Output) StackOutputInfos {
	spi := make(StackOutputInfos, len(outputs))
	for i, p := range outputs {
		spi[i] = newStackOutputInfo(p)
	}
	return spi
}

func (spi StackOutputInfos) String() string {
	var buffer bytes.Buffer

	data := make([][]string, len(spi))

	for i, p := range spi {
		data[i] = []string{"", bold(p.Key), cyan(p.Value)}
	}

	table := tablewriter.NewWriter(&buffer)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()

	return buffer.String()
}

// StackOutputInfo represents a Stack's output
type StackOutputInfo struct {
	Key   string
	Value string
}

func newStackOutputInfo(output *cf.Output) StackOutputInfo {
	return StackOutputInfo{
		Key:   deref(output.OutputKey),
		Value: deref(output.OutputValue),
	}
}

// StackInfo represents the state of a stack in Cloudformation
type StackInfo struct {
	ID              string
	Name            string
	Status          string
	CreationTime    time.Time
	LastUpdatedTime time.Time
	Params          StackParamInfos
	Outputs         StackOutputInfos
}

func newStackInfo(stack *cf.Stack) *StackInfo {
	si := &StackInfo{
		ID:              deref(stack.StackId),
		Name:            *stack.StackName,
		Status:          *stack.StackStatus,
		CreationTime:    *stack.CreationTime,
		LastUpdatedTime: *stack.CreationTime,
		Params:          newStackParamInfos(stack.Parameters),
		Outputs:         newStackOutputInfos(stack.Outputs),
	}

	if stack.LastUpdatedTime != nil {
		si.LastUpdatedTime = *stack.LastUpdatedTime
	}
	return si
}

func (si *StackInfo) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("%s: %s\n", underline(bold("Stack")), cyan(si.Name)))

	data := [][]string{
		{bold("ID"), cyan(si.ID)},
		{bold("Status"), cyan(si.Status)},
		{bold("CreationTime"), cyan(si.CreationTime)},
		{bold("LastUpdatedTime"), cyan(si.LastUpdatedTime)},
	}

	table := tablewriter.NewWriter(&buffer)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	table.SetAutoMergeCells(true)
	table.SetAutoWrapText(false)
	table.AppendBulk(data)
	table.Render()

	buffer.WriteString(fmt.Sprintf("  %s\n", bold("Params")))
	buffer.WriteString(si.Params.String())

	buffer.WriteString(fmt.Sprintf("  %s\n", bold("Outputs")))
	buffer.WriteString(si.Outputs.String())

	return buffer.String()
}

// CanUpdate indicates whether a stack can be updated
func (si *StackInfo) CanUpdate() bool {
	switch si.Status {
	case cf.StackStatusReviewInProgress:
		return false
	default:
		return true
	}
}

func deref(ptrStr *string) string {
	if ptrStr == nil {
		return ""
	}
	return *ptrStr
}
