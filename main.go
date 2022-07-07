package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
)

func main() {
	var app App

	ctx := context.Background()
	switch err := app.Run(ctx); err {
	case errSameWindow:
		os.Exit(1)
	case nil:
	default:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

var errSameWindow = fmt.Errorf("same window")

type App struct {
	Dir     string
	Addr    string
	Action  string
	Timeout time.Duration
}

func (a *App) Init(ctx context.Context) (context.Context, context.CancelFunc, error) {
	cancel := func() {}

	flags := flag.NewFlagSet("nvim-wezterm", flag.ContinueOnError)

	flags.StringVar(&a.Dir, "dir", "", "h, j, k, or l: direction to move in neovim")
	flags.StringVar(&a.Addr, "addr", "", "address neovim is listening on")
	flags.StringVar(&a.Action, "action", "", "move or resize")
	flags.DurationVar(&a.Timeout, "timeout", 25*time.Millisecond, "timeout for connection")

	if err := flags.Parse(os.Args[1:]); err != nil {
		flags.Usage()
		return ctx, cancel, err
	}

	if a.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, a.Timeout)
	}

	if a.Addr == "" {
		flags.Usage()
		return ctx, cancel, fmt.Errorf("missing -addr")
	}

	if _, err := os.Stat(a.Addr); err != nil {
		flags.Usage()
		return ctx, cancel, err
	}

	a.Dir = strings.ToLower(a.Dir)

	switch a.Action {
	case "move", "resize":
		switch a.Dir {
		case "h", "j", "k", "l":
		default:
			flags.Usage()
			return ctx, cancel, fmt.Errorf("invalid -dir: %q", a.Dir)
		}
	default:
		flags.Usage()
		return ctx, cancel, fmt.Errorf("invalid -action: %q", a.Action)
	}

	return ctx, cancel, nil
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel, err := a.Init(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	opt := nvim.DialContext(ctx)
	c, err := nvim.Dial(a.Addr, opt)
	if err != nil {
		return err
	}

	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)

		switch a.Action {
		case "move":
			err = a.Move(c)
		case "resize":
			err = a.Resize(c)
		}
	}()

	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *App) Move(c *nvim.Nvim) error {
	b := c.NewBatch()

	var curWinNr int
	b.WindowNumber(0, &curWinNr)

	var nextWinNr int
	b.Eval(fmt.Sprintf("winnr('%s')", a.Dir), &nextWinNr)

	if err := b.Execute(); err != nil {
		return err
	}

	if nextWinNr == curWinNr {
		return errSameWindow
	}

	return nil
}

func (a *App) Resize(c *nvim.Nvim) error {
	var (
		winNrs [2]int
		dirs   [2]string
	)

	switch a.Dir {
	case "h", "l": // width
		dirs = [2]string{"h", "l"}
	case "j", "k": // height
		dirs = [2]string{"j", "k"}
	}

	b := c.NewBatch()

	var curWinNr int
	b.WindowNumber(0, &curWinNr)

	for i, dir := range dirs {
		b.Eval(fmt.Sprintf("winnr('%s')", dir), &winNrs[i])
	}

	if err := b.Execute(); err != nil {
		return err
	}

	for _, winNr := range winNrs {
		if curWinNr != winNr {
			return nil
		}
	}

	return errSameWindow
}
