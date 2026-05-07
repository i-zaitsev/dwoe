package commands

import (
	"flag"
	"log/slog"

	"github.com/i-zaitsev/dwoe/internal/cli"
)

type cmdRefine struct {
	parentNameOrID string
	name           string
	do             string
	work           string
	detach         bool
}

func (c *cmdRefine) Name() string { return "refine" }
func (c *cmdRefine) Desc() string { return "Refine a completed workspace with a new prompt" }
func (c *cmdRefine) Args() string { return "<parent-name|id> --do <prompt> | --work <path>" }

func (c *cmdRefine) Parse(args []string) error {
	fs, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.do, "do", "", "inline task prompt")
		fs.StringVar(&c.do, "d", "", "inline task prompt")
		fs.StringVar(&c.work, "work", "", "path to work file")
		fs.StringVar(&c.work, "w", "", "path to work file")
		fs.StringVar(&c.name, "name", "", "override workspace name")
		fs.BoolVar(&c.detach, "detach", false, "start and return immediately")
		fs.BoolVar(&c.detach, "D", false, "start and return immediately")
	})
	if err != nil {
		return err
	}
	if c.do != "" && c.work != "" {
		return cli.CmdErr(c, "cannot use both --do and --work")
	}
	if fs.NArg() == 0 {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "parent workspace"})
	}
	c.parentNameOrID = fs.Arg(0)
	if c.do == "" && c.work == "" {
		return cli.CmdErr(c, "either --do or --work is required")
	}
	return nil
}

func (c *cmdRefine) Run(e *cli.Env) error {
	slog.Info("cli: refine", "parent", c.parentNameOrID, "detach", c.detach)

	prompt := c.do
	if c.work != "" {
		resolved, err := resolveWork(c.work)
		if err != nil {
			return cli.CmdErr(c, "%w", err)
		}
		prompt = resolved
	}

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	child, err := manager.Refine(c.parentNameOrID, prompt, c.name)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ctx := e.Context()
	if err := manager.Start(ctx, child.ID); err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	e.Print("Refining workspace: %s\n", child.Name)
	e.Print("- ID: %s\n", child.ID)
	e.Print("- Parent: %s\n", c.parentNameOrID)
	e.Print("- Status: running\n")
	e.Print("- Path: %s\n", child.BasePath)

	if c.detach {
		e.Print("View logs: %s logs %s\n", cli.Prog, child.ID)
		return nil
	}

	if err := runAttached(e, manager, child.ID, child.Name); err != nil {
		return cli.CmdErr(c, "%w", err)
	}
	return nil
}
