package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errBadNumberOfArguments = errors.New("bad number of arguments")
	errBadArgument          = errors.New("bad argument")
	errOutOfRange           = errors.New("out of range")
)

func digits(x int) int {
	n := 0
	for x > 0 {
		n++
		x /= 10
	}
	return n
}

var (
	shellCommandsMu sync.RWMutex
	shellCommands   = make(map[string]bool)
)

func lookupShellCommand(name string) bool {
	shellCommandsMu.RLock()
	isShellCommand, ok := shellCommands[name]
	shellCommandsMu.RUnlock()
	if ok {
		return isShellCommand
	}
	_, err := exec.LookPath(name)
	isShellCommand = err == nil
	shellCommandsMu.Lock()
	defer shellCommandsMu.Unlock()
	shellCommands[name] = isShellCommand
	return isShellCommand
}

type command struct {
	name        string
	aliases     []string
	format      string
	usage       string
	appendStdin bool
	// complete callback used to auto complete for command.
	// args parsed from input[:pos], and input[pos:] is the
	// unparsed part. returned strings would be appended to input.
	complete func(input string, pos int, args []string) []string
	run      func(ctx context.Context, env *enviroment, args []string) error
}

var (
	commands     = make(map[string]*command)
	uniqCommands []*command
	complet      completion
)

func register(cmd *command) *command {
	if _, dup := commands[cmd.name]; dup {
		panic("command " + cmd.name + " duplicated")
	}
	for _, alias := range cmd.aliases {
		if _, dup := commands[alias]; dup {
			panic("command alias " + alias + " duplicated")
		}
	}
	commands[cmd.name] = cmd
	complet.add(cmd.name)
	for _, alias := range cmd.aliases {
		commands[alias] = cmd
		complet.add(alias)
	}
	uniqCommands = append(uniqCommands, cmd)
	return cmd
}

var cmdExit = register(&command{
	name:    "exit",
	aliases: []string{"quit"},
	usage:   "exit the terminal",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		return nil
	},
})

var cmdHelp = register(&command{
	name:    "help",
	aliases: []string{"h"},
	usage:   "show help information",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		var (
			all              bool
			sorted           []*command
			notFoundCommands int
		)
		if len(args) > 0 {
			for i := range args {
				cmd, ok := commands[args[i]]
				if ok {
					if notFoundCommands == 0 {
						sorted = append(sorted, cmd)
					}
				} else {
					isRedisCommand, isShellCommand := env.find(args[i])
					if isRedisCommand && !isShellCommand {
						sorted = append(sorted, &command{
							name:  args[i],
							usage: args[i] + " is a redis command",
						})
					} else if !isRedisCommand && isShellCommand {
						sorted = append(sorted, &command{
							name:  args[i],
							usage: args[i] + " is a shell command",
						})
					} else if isRedisCommand && isShellCommand {
						sorted = append(sorted, &command{
							name:  args[i],
							usage: args[i] + " is a redis/shell command",
						})
					} else {
						notFoundCommands++
						env.errorln("command %q not found", args[i])
					}
				}
			}
			if notFoundCommands > 0 {
				return nil
			}
		} else {
			all = true
			for _, cmd := range uniqCommands {
				sorted = append(sorted, cmd)
			}
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].name < sorted[j].name
		})
		for _, cmd := range sorted {
			var aliases string
			if len(cmd.aliases) > 0 {
				aliases = " (alias: " + strings.Join(cmd.aliases, ",") + ")"
			}
			if cmd.format == "" {
				env.println("%s%s", cmd.name, aliases)
			} else {
				env.println("%s %s%s", cmd.name, cmd.format, aliases)
			}
			usage := strings.Split(cmd.usage, "\n")
			for i := range usage {
				env.println("\t%s", usage[i])
			}
		}
		if all {
			env.println("")
			env.println("All redis commands supported after connected to redis server.")
			env.println("Using 'sh ...' execute local shell commands, e.g. sh echo xxx.")
			env.println("Chain calls supported via pipeline '|', e.g. xgrep xmap.xmonster | less.")
		}
		return nil
	},
})

var cmdHistory = register(&command{
	name:   "history",
	format: "[limit]",
	usage:  "show history commands",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if len(args) > 1 {
			env.errorln(errBadNumberOfArguments.Error())
			return errBadNumberOfArguments
		}
		limit := -1
		if len(args) == 1 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				env.errorln("numeric argument required")
				return errBadArgument
			}
			limit = n
		}
		all := env.history.all()
		if limit < 0 || limit >= len(all) {
			limit = 0
		} else {
			limit = len(all) - limit
		}
		if limit == len(all) {
			return nil
		}
		format := "%" + strconv.Itoa(digits(len(all)-limit)) + "d %s"
		for i, h := range all[limit:] {
			env.println(format, i+1, h)
		}
		return nil
	},
})

var cmdRun = register(&command{
	name:   "run",
	format: "<file> [more files...]",
	usage:  "exec script files",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if len(args) == 0 {
			env.errorln("missing filename")
			return errBadNumberOfArguments
		}
		for i := range args {
			var filename = args[i]
			if !filepath.IsAbs(filename) {
				dir, _ := filepath.Split(filename)
				if strings.HasPrefix(dir, ".") {
					filename = filepath.Join(env.cwd, filename)
				} else {
					filename = filepath.Join(env.pwd, filename)
				}
			}
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				env.errorln("load script %q error: %v", args[i], err)
				return err
			}
			s := bufio.NewScanner(bytes.NewReader(content))
			var (
				fork *enviroment
				cwd  = env.cwd
			)
			if env.nofork {
				fork = env
			} else {
				fork = env.fork()
				fork.stderr = env.stderr
			}
			fork.cwd = filepath.Dir(filename)
			var broken bool
		SCAN:
			for s.Scan() {
				line := s.Text()
				fork.run(ctx, line, env.stdin, env.stdout)
				if fork.exitCode != 0 && fork.scriptOptions.e {
					break SCAN
				}
				select {
				case <-ctx.Done():
					broken = true
					break SCAN
				default:
				}
			}
			env.cwd = cwd
			if !broken {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
			}
		}
		return nil
	},
})

var cmdPause = register(&command{
	name:   "pause",
	format: "[N]",
	usage:  "pause N seconds, if N <= 0, pause forever until received an INT signal",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		var milliseconds int
		if len(args) == 1 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				env.errorln("numeric argument required")
				return errBadArgument
			}
			milliseconds = n * 1000
		} else if len(args) > 1 {
			env.errorln(errBadNumberOfArguments.Error())
			return errBadNumberOfArguments
		}
		if milliseconds > 0 {
			env.println("paused %d seconds", milliseconds/1000)
			if milliseconds <= 5000 {
				time.Sleep(time.Duration(milliseconds) * time.Millisecond)
			} else {
				const length = 50
				n := milliseconds / length
				env.println("|%s|", strings.Repeat("-", length))
				env.printf("|")
				ticker := time.NewTicker(time.Duration(n) * time.Millisecond)
				defer ticker.Stop()
				var broken bool
			WAIT:
				for {
					select {
					case <-ticker.C:
						env.printf(".")
						milliseconds -= n
						if milliseconds < n {
							break WAIT
						}
					case <-ctx.Done():
						broken = true
						if err := ctx.Err(); err != nil {
							env.println("canceled: %v", err)
						} else {
							env.println("canceled")
						}
						break WAIT
					}
				}
				if !broken {
					if milliseconds > 0 {
						time.Sleep(time.Duration(milliseconds) * time.Millisecond)
					}
					env.println("| 100%")
				}
			}
			return nil
		}
		env.println("paused forever until received an INT signal")
		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch, os.Interrupt)
	WAIT_FOREVER:
		for {
			select {
			case sig := <-sigch:
				if sig == os.Interrupt {
					env.println("resumed")
					break WAIT_FOREVER
				}
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					env.println("canceled: %v", err)
				} else {
					env.println("canceled")
				}
				break WAIT_FOREVER
			}
		}
		signal.Stop(sigch)
		close(sigch)
		return nil
	},
})

var cmdLogcat = register(&command{
	name:   "logcat",
	format: "[lines=1000]",
	usage:  "show logfile tail content",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if len(args) > 1 {
			env.errorln(errBadNumberOfArguments.Error())
			return errBadNumberOfArguments
		}
		lines := "1000"
		if len(args) > 0 {
			_, err := strconv.Atoi(args[0])
			if err != nil {
				env.errorln("numeric argument required")
				return err
			}
			lines = args[0]
		}
		return cmdShell.run(ctx, env, []string{"tail", "-" + lines, logfile()})
	},
})

var cmdSetE = register(&command{
	name:  "set+e",
	usage: "set flags to interrupt script for error",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		env.scriptOptions.e = true
		return nil
	},
})

var cmdUnsetE = register(&command{
	name:  "set-e",
	usage: "unset flags to interrupt script for error",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		env.scriptOptions.e = false
		return nil
	},
})
