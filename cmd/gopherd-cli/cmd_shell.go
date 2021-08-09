package main

import (
	"context"
	"errors"
	"os/exec"
)

var cmdShell = register(&command{
	name:    "shell",
	aliases: []string{"sh"},
	format:  "<shell command>",
	usage:   "exec shell command",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if len(args) == 0 {
			err := errors.New("command name required")
			env.errorln(err.Error())
			return err
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdin = env.stdin
		cmd.Stdout = env.stdout
		cmd.Stderr = env.stderr
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				env.exitCode = ee.ExitCode()
			}
		}
		return nil
	},
})
