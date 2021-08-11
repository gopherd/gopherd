package main

import (
	"context"
	"fmt"
)

var cmdTypes = register(&command{
	name:  "types",
	usage: "show all message types",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		return nil
	},
})

var cmdSend = register(&command{
	name:        "send",
	format:      "-type TYPE [content or stdin]",
	usage:       "send message to gate",
	appendStdin: true,
	run: func(ctx context.Context, env *enviroment, args []string) error {
		var flags struct {
			messageType int
		}
		flagSet := env.newFlagSet()
		flagSet.IntVar(*flags.messageType, "type", 0, "message type")
		if flags.messageType <= 0 {
			err := fmt.Errorf("message type MUST be greater than 0, but got %d", flags.messageType)
			env.errorln(err.Error())
			return err
		}
		args = flagSet.Args()
		if len(args) != 1 {
			err = errBadNumberOfArguments
			env.errorln(err.Error())
			return err
		}
		return nil
	},
})
