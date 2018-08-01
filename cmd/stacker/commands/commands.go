package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eyeamera/stacker-cli/client"
	"github.com/fatih/color"
	"github.com/jawher/mow.cli"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

var (
	bold      = color.New(color.Bold).SprintFunc()
	underline = color.New(color.Underline).SprintFunc()
	cyan      = color.New(color.FgCyan).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
)

func newStackerClient(region string) *client.Client {
	return client.New(client.NewCloudformationClient(region))
}

type Backend interface {
	Fetch(name string) ([]client.Stack, error)
}

func Update(b Backend) func(cmd *cli.Cmd) {
	return func(cmd *cli.Cmd) {
		var (
			stack            client.Stack
			stacker          *client.Client
			stackName        = cmd.StringArg("STACK", "", "Stack name")
			allowDestructive = cmd.Bool(cli.BoolOpt{
				Name:  "y allow-destructive",
				Value: false,
				Desc:  "Allow destructive changes",
			})
		)

		cmd.Spec = "STACK [-y] [--allow-destructive]"

		cmd.Before = func() {
			stack = fetchStack(b, *stackName)
			stacker = newStackerClient(stack.Region())
		}

		cmd.Action = func() {
			cs, err := plan(stacker, stack)
			if err != nil {
				exitWithError(err)
			}

			review(stacker, cs)

			if !confirmChanges(cs, *allowDestructive) {
				os.Exit(1)
			}

			if err := apply(stacker, cs); err != nil {
				exitWithError(err)
			}

			fmt.Println(bold("Stack update completed successfully"))
		}
	}
}

func Plan(b Backend) func(cmd *cli.Cmd) {
	return func(cmd *cli.Cmd) {
		var (
			stack     client.Stack
			stacker   *client.Client
			stackName = cmd.StringArg("STACK", "", "Stack name")
		)

		cmd.Spec = "STACK"

		cmd.Before = func() {
			stack = fetchStack(b, *stackName)
			stacker = newStackerClient(stack.Region())
		}

		cmd.Action = func() {
			cs, err := plan(stacker, stack)
			if err != nil {
				exitWithError(err)
			}

			if !cs.CanCommit() {
				return
			}

			fmt.Printf(
				"  %s: `%s`\n",
				bold("Review these changes with"),
				cyan(fmt.Sprintf("stacker review %s %s", cs.StackName, cs.Name)),
			)

			fmt.Printf(
				"  %s: `%s`\n\n",
				bold("Apply these changes with"),
				cyan(fmt.Sprintf("stacker apply %s %s", cs.StackName, cs.Name)),
			)
		}
	}
}

func Review(b Backend) func(cmd *cli.Cmd) {
	return func(cmd *cli.Cmd) {
		var (
			stack     client.Stack
			stacker   *client.Client
			stackName = cmd.StringArg("STACK", "", "Stack name")
			changeSet = cmd.StringArg("CHANGESET", "", "Changeset name")
		)

		// @TODO Allow stack to not exist locally for this

		cmd.Spec = "STACK [CHANGESET]"

		cmd.Before = func() {
			stack = fetchStack(b, *stackName)
			stacker = newStackerClient(stack.Region())
			ensureStackExists(stacker, *stackName)
		}

		cmd.Action = func() {
			cs, err := fetchChangeSet(stacker, *stackName, *changeSet)
			if err != nil {
				exitWithError(err)
			}

			review(stacker, cs)

			if !cs.CanCommit() {
				return
			}

			fmt.Printf(
				"  %s: `%s`\n\n",
				bold("Apply these changes with"),
				cyan(fmt.Sprintf("stacker apply %s %s", cs.StackName, cs.Name)),
			)
		}
	}
}

func Apply(b Backend) func(cmd *cli.Cmd) {
	return func(cmd *cli.Cmd) {
		var (
			stack            client.Stack
			stacker          *client.Client
			stackName        = cmd.StringArg("STACK", "", "Stack name")
			changeSet        = cmd.StringArg("CHANGESET", "", "Changeset name")
			allowDestructive = cmd.Bool(cli.BoolOpt{
				Name:  "y allow-destructive",
				Value: false,
				Desc:  "Allow destructive changes",
			})
		)

		cmd.Spec = "STACK [CHANGESET] [-y] [--allow-destructive]"

		// @TODO Allow stack to not exist locally for this

		cmd.Before = func() {
			stack = fetchStack(b, *stackName)
			stacker = newStackerClient(stack.Region())
			ensureStackExists(stacker, *stackName)
		}

		cmd.Action = func() {
			cs, err := fetchChangeSet(stacker, *stackName, *changeSet)
			if err != nil {
				exitWithError(err)
			}

			review(stacker, cs)

			if !confirmChanges(cs, *allowDestructive) {
				os.Exit(1)
			}

			if err := apply(stacker, cs); err != nil {
				exitWithError(err)
			}

			fmt.Println(bold("Stack update completed successfully"))

			fmt.Printf(
				"\n  %s: `%s`\n\n",
				bold("View stack status with"),
				cyan(fmt.Sprintf("stacker show %s", cs.StackName)),
			)
		}
	}
}

func Delete(b Backend) func(cmd *cli.Cmd) {
	return func(cmd *cli.Cmd) {
		var (
			stack     client.Stack
			stacker   *client.Client
			stackName = cmd.StringArg("STACK", "", "Stack name")
		)

		// @TODO Allow stack to not exist locally for this

		cmd.Spec = "STACK"

		cmd.Before = func() {
			stack = fetchStack(b, *stackName)
			stacker = newStackerClient(stack.Region())
			ensureStackExists(stacker, *stackName)
		}

		cmd.Action = func() {
			fmt.Printf("%s: %s\n\n", bold("Deleting stack"), cyan(*stackName))
			fmt.Printf("  %s\n\n",
				underline("This is a destructive action and will delete your stack and all of its associated resources!"),
			)

			if confirm(bold("  Enter stack name to continue: ")) != *stackName {
				exitWithError(errors.New("deletion must be confirmed with stack name"))
			}

			fmt.Println()

			if err := delete(stacker, *stackName); err != nil {
				exitWithError(err)
			}
		}
	}
}

func Show(b Backend) func(cmd *cli.Cmd) {
	return func(cmd *cli.Cmd) {
		var (
			stack     client.Stack
			stacker   *client.Client
			stackName = cmd.StringArg("STACK", "", "Stack name")
		)

		cmd.Spec = "STACK"

		cmd.Before = func() {
			stack = fetchStack(b, *stackName)
			stacker = newStackerClient(stack.Region())
			ensureStackExists(stacker, *stackName)
		}

		cmd.Action = func() {
			if err := show(stacker, *stackName); err != nil {
				exitWithError(err)
			}
		}
	}
}

func fetchStack(b Backend, name string) client.Stack {
	s, err := b.Fetch(name)
	if err != nil {
		exitWithError(err)
	}

	if len(s) > 1 {
		exitWithError(fmt.Errorf("multiple stacks found for `%s`", name))
	}

	if len(s) == 0 {
		exitWithError(fmt.Errorf("no stack found for `%s`", name))
	}

	return s[0]
}

// Plan creates a new changeset given a client and a stack
func plan(stacker *client.Client, stack client.Stack) (*client.ChangeSetInfo, error) {
	var (
		si  *client.StackInfo
		cs  *client.ChangeSetInfo
		err error
	)

	if si, err = stacker.Get(stack.Name()); err != nil {
		return nil, errors.Wrap(err, "failed to fetch stack information")
	}

	if si == nil {
		fmt.Printf("%s %s\n", bold("Creating changeset for new stack"), cyan(stack.Name()))
		cs, err = stacker.Create(stack)
	} else {
		if !si.CanUpdate() {
			return nil, errors.Errorf(
				"cannot update %s at this time. stack status=%s",
				stack.Name(),
				si.Status,
			)
		}
		fmt.Printf("%s %s\n", bold("Creating changeset to update stack"), cyan(stack.Name()))
		cs, err = stacker.Update(stack)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to create new changeset")
	}

	fmt.Printf("%s: %s\n", bold("Changeset created"), cyan(cs.Name))
	fmt.Printf("%s... %s\n", bold("Waiting for changeset to complete creation"), "use ^C to exit safely")

	stacker.WaitForChangeSetComplete(cs.StackName, cs.Name)

	fmt.Printf("%s.\n\n", bold("Changeset creation complete"))

	return stacker.GetChangeSet(cs.StackName, cs.Name)
}

// Review displays information about a changeset
func review(stacker *client.Client, changeSet *client.ChangeSetInfo) {
	fmt.Println(changeSet)

	stackInfo, err := stacker.Get(changeSet.StackName)
	if err != nil {
		exitWithError(fmt.Errorf("error fetching information for stack %s", changeSet.StackName))
	}

	reviewStackParams(changeSet.Params, stackInfo.Params)
}

// Apply executes a changeset against a stack
func apply(stacker *client.Client, changeSet *client.ChangeSetInfo) error {
	if !changeSet.CanCommit() {
		return errors.Errorf("change set %s cannot be applied. status=%s", changeSet.Name, changeSet.Status)
	}

	fmt.Printf("%s %s %s %s\n", bold("Applying changeset"), cyan(changeSet.Name), bold("to"), cyan(changeSet.StackName))

	if err := stacker.Commit(changeSet.StackName, changeSet.Name); err != nil {
		return errors.Wrapf(err, "error committing changeset %s", changeSet.Name)
	}

	fmt.Printf("%s... %s\n\n", bold("Waiting for changeset to apply"), "use ^C to exit safely")

	return stacker.NotifyUntilComplete(changeSet.StackName, showStackEvents(stacker))
}

// Show prints information about a stack
func show(stacker *client.Client, stackName string) error {
	var (
		stackInfo    *client.StackInfo
		resourceInfo client.ResourceInfos
		events       client.StackEvents
		err          error
	)

	if stackInfo, err = stacker.Get(stackName); err != nil {
		return errors.Wrapf(err, "error fetching stack %s", stackName)
	}

	fmt.Println(stackInfo)

	if resourceInfo, err = stacker.GetResources(stackName); err != nil {
		return errors.Wrapf(err, "error fetching stack %s resources", stackName)
	}

	if len(resourceInfo) > 0 {
		fmt.Printf("%s:\n", bold("Resources"))
		fmt.Println(resourceInfo)
	}

	if events, err = stacker.GetEvents(stackName); err != nil {
		return errors.Wrapf(err, "error fetching stack %s events", stackName)
	}

	if len(events) > 0 {
		fmt.Printf("%s:\n", bold("Events"))
		fmt.Println(events)
	}

	return nil
}

// Delete removes a stack
func delete(stacker *client.Client, stackName string) error {
	fmt.Printf("%s %s\n", bold("Deleting stack"), cyan(stackName))

	if err := stacker.Delete(stackName); err != nil {
		return errors.Wrapf(err, "error deleting stack %s", stackName)
	}

	fmt.Printf("%s... %s\n", bold("Waiting for stack to complete deletion"), "use ^C to exit safely")

	return stacker.NotifyUntilComplete(stackName, showStackEvents(stacker))
}

// fetchChangeSet fetches the ChangeSetInfo provided a stackName and changeSetName.
// It will interactively prompt the user to select a changeset in the event that multiple
// changesets exist when provided an empty changeSetName param.
func fetchChangeSet(stacker *client.Client, stackName string, changeSetName string) (*client.ChangeSetInfo, error) {
	if changeSetName != "" {
		return stacker.GetChangeSet(stackName, changeSetName)
	}

	pcs, err := stacker.GetChangeSets(stackName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch available changesets")
	}

	if len(pcs) == 0 {
		return nil, errors.Errorf("no changesets found for %s", stackName)
	}

	if len(pcs) == 1 {
		return stacker.GetChangeSet(stackName, pcs[0].Name)
	}

	fmt.Printf("\n%s\n\n", bold("Select a changeset:"))

	data := make([][]string, len(pcs))
	for i, p := range pcs {
		data[i] = []string{
			bold(strconv.Itoa(i + 1)),
			bold(cyan(p.Name)),
			bold(p.Status),
			p.CreationTime.String(),
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.AppendBulk(data)
	table.Render()

	fmt.Println("")

	for {
		fmt.Print("Changeset [1]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			changeSetName = pcs[0].Name
			break
		}

		index, err := strconv.Atoi(input)
		if err != nil || index > len(pcs) || index < 1 {
			continue
		}

		changeSetName = pcs[index-1].Name
		break
	}

	fmt.Println("")

	return stacker.GetChangeSet(stackName, changeSetName)
}

// Prompts when a changeset will modify or remove resources.
func confirmChanges(cs *client.ChangeSetInfo, allowDestructive bool) bool {
	if !changeSetHasChanges(cs) {
		return true
	}

	if changeSetIsDestructive(cs) && !allowDestructive {
		fmt.Printf("  %s\n\n",
			underline("This is a destructive action and will replace or delete stack resources!"),
		)
		if confirm(bold("  Proceed with changes (y/n)?: ")) != "y" {
			return false
		}
		fmt.Println()
	} else {
		fmt.Printf("  %s\n\n",
			underline("This action will modify or remove stack resources."),
		)
		if confirm(bold("  Proceed with changes (y/n)?: ")) != "y" {
			return false
		}
		fmt.Println()
	}

	return true
}

func showStackEvents(stacker *client.Client) func(s *client.StackInfo) {
	cursor := time.Now()
	return func(s *client.StackInfo) {
		events, err := stacker.GetEvents(s.Name)
		if err != nil {
			fmt.Println(red("Error fetching stack events"))
		}
		for i := len(events) - 1; i >= 0; i-- {
			e := events[i]
			// fmt.Println(cursor, e.Timestamp)
			if cursor.Before(e.Timestamp) {
				fmt.Printf("%s %s\n  %s %s (%s)\n  %s\n\n",
					e.Timestamp,
					bold(e.Resource.Status),
					underline(bold(e.Resource.Type)),
					cyan(e.Resource.Name),
					cyan(e.Resource.ID),
					cyan(e.Resource.StatusReason),
				)
			}
		}
		cursor = events[0].Timestamp
	}
}

func reviewStackParams(local client.StackParamInfos, remote client.StackParamInfos) {
	allKeys := make(map[string]bool)

	remoteMap := make(map[string]client.StackParamInfo)
	for _, p := range remote {
		remoteMap[p.Key] = p
		allKeys[p.Key] = true
	}

	localMap := make(map[string]client.StackParamInfo)
	for _, p := range local {
		localMap[p.Key] = p
		allKeys[p.Key] = true
	}

	data := make([][]string, len(allKeys))
	data = append(data, []string{
		"", bold("changeset"), bold("stack"),
	})
	for k, _ := range allKeys {

		l := localMap[k].Value
		if l == "" {
			l = "<notset>"
		}

		r := remoteMap[k].Value
		if r == "" {
			r = "<notset>"
		}

		c := cyan
		if l != r {
			c = yellow
		}

		data = append(data, []string{
			bold(k), c(l), c(r),
		})
	}

	var buffer bytes.Buffer

	table := tablewriter.NewWriter(&buffer)
	table.SetColumnSeparator("")
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()

	fmt.Printf("%s\n%s\n", bold(underline("Stack Params:")), buffer.String())
}

func changeSetHasChanges(changeSet *client.ChangeSetInfo) bool {
	for _, c := range changeSet.Changes {
		if c.Action == "Modify" || c.Action == "Remove" {
			return true
		}
	}

	return false
}

// Determines if a changeset will cause a destructive change
func changeSetIsDestructive(changeSet *client.ChangeSetInfo) bool {
	for _, c := range changeSet.Changes {
		if c.Action == "Modify" && c.Replacement {
			return true
		}

		if c.Action == "Remove" {
			return true
		}
	}

	return false
}

// confirm prompts the user with input field, and returns
// the user input
func confirm(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func ensureStackExists(stacker *client.Client, stackName string) {
	exists, err := stacker.Exists(stackName)
	if err != nil {
		exitWithError(err)
	}

	if !exists {
		exitWithError(errors.Errorf("stack %s does not exist", stackName))
	}
}

func exitWithError(err error) {
	fmt.Println(bold(red(err)))
	cli.Exit(1)
}
