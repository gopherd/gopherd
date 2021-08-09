package main

import (
	"context"
	"strconv"
	"strings"
)

var cmdLine = register(&command{
	name:        "line",
	format:      "<i> [content | stdin]",
	usage:       "get i-th (beginning 0) line",
	appendStdin: true,
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if len(args) != 2 {
			env.errorln(errBadNumberOfArguments.Error())
			return errBadNumberOfArguments
		}
		i, err := strconv.Atoi(args[0])
		if err != nil {
			env.errorln("bad index: %v", err)
			return err
		}
		lines := strings.Split(args[1], "\n")
		if i < -len(lines) || i >= len(lines) {
			env.errorln("index %d out of range [%d, %d]", i, -len(lines), len(lines)-1)
			return errOutOfRange
		}
		if i < 0 {
			i += len(lines)
		}
		env.println(lines[i])
		return nil
	},
})

var cmdAt = register(&command{
	name:   "at",
	format: "<i> [content | stdin]",
	usage: "get i-th (beginning 0) string from strings splitted by space, e.g.\n" +
		"\techo xx yy zz | at 1   # output: yy\n" +
		"\techo xx yy zz | at -1  # output: zz",
	appendStdin: true,
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if len(args) != 2 {
			env.errorln(errBadNumberOfArguments.Error())
			return errBadNumberOfArguments
		}
		i, err := strconv.Atoi(args[0])
		if err != nil {
			env.errorln("bad index: %v", err)
			return err
		}
		lines := strings.Split(args[1], " ")
		for i := len(lines) - 1; i >= 0; i-- {
			if lines[i] == "" {
				lines = append(lines[:i], lines[i+1:]...)
			}
		}
		if i < -len(lines) || i >= len(lines) {
			env.errorln("index %d out of range [%d, %d]", i, -len(lines), len(lines)-1)
			return errOutOfRange
		}
		if i < 0 {
			i += len(lines)
		}
		env.printf(lines[i])
		return nil
	},
})

var cmdUnquote = register(&command{
	name:        "unquote",
	aliases:     []string{"uq"},
	format:      "[content | stdin]",
	usage:       "unquote the quoted string",
	appendStdin: true,
	run: func(ctx context.Context, env *enviroment, args []string) error {
		n := 0
		for i := range args {
			s, err := strconv.Unquote(args[i])
			if err != nil {
				env.errorln(err.Error())
				return err
			} else {
				if n > 0 {
					env.printf(" %s", s)
				} else {
					env.printf("%s", s)
				}
				n++
			}
		}
		return nil
	},
})
