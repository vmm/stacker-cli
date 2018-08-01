package main

import (
	"os"

	"github.com/eyeamera/stacker-cli/backend"
	"github.com/eyeamera/stacker-cli/cmd/stacker/commands"
	cli "github.com/jawher/mow.cli"
)

func main() {
	stackerPath := "."
	if path := os.Getenv("STACKER_PATH"); path != "" {
		stackerPath = path
	}

	b := backend.New(stackerPath)

	app := cli.App("stacker", "Manage Cloudformation Stacks")

	app.Command("list", "List available stacks", commands.List(b))

	// Require a stack
	app.Command("show", "Show information about a stack", commands.Show(b))
	app.Command("plan", "Plan a change to a stack by creating a changeset", commands.Plan(b))
	app.Command("review", "Review a changeset", commands.Review(b))
	app.Command("apply", "Apply a changeset", commands.Apply(b))
	app.Command("update", "Update performs a plan, review and an apply on a stack", commands.Update(b))
	app.Command("delete", "Delete a stack", commands.Delete(b))

	app.Run(os.Args)
}
