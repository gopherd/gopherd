package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/gizak/termui/v3"
	"github.com/go-redis/redis/v8"
	"github.com/gopherd/doge/text/shell"
	"github.com/mattn/go-isatty"
)

func toExitError(e error) (err error, ee *exitError, ok bool) {
	if e == nil {
		return nil, nil, false
	}
	err = e
	for {
		if ee, ok = e.(*exitError); ok {
			return
		}
		if e = errors.Unwrap(e); e == nil {
			break
		}
	}
	return
}

func readAll(r io.Reader, b64 bool) ([]byte, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if b64 {
		decoded, err := base64.StdEncoding.DecodeString(string(content))
		if err != nil {
			return nil, err
		}
		content = decoded
	}
	return content, nil
}

var config struct {
	Unsafe bool   `json:"unsafe"`
	Redis  string `json:"redis"`
}

type forkable struct {
	unsafe bool
	redis  *redisClient

	pwd string
	cwd string

	history *stRingBuffer
	logfile io.WriteCloser

	scriptOptions struct {
		e bool // set-e
	}
}

type redisClient struct {
	conn     *redis.Client
	commands map[string]*redis.CommandInfo
}

type enviroment struct {
	forkable
	nofork bool
	forked bool

	more struct {
		content bytes.Buffer
		size    int
		what    string
	}

	name     string
	exitCode int
	stdin    io.Reader

	wmu    sync.Mutex
	stdout io.Writer
	stderr io.Writer
}

func newEnviroment() *enviroment {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "."
	}
	history := new(stRingBuffer)
	history.init()
	return &enviroment{
		forkable: forkable{
			pwd:     pwd,
			cwd:     pwd,
			redis:   new(redisClient),
			history: history,
		},
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func (env *enviroment) fork() *enviroment {
	fork := new(enviroment)
	fork.forkable = env.forkable
	fork.forked = true
	return fork
}

func (env *enviroment) redisc() *redis.Client {
	return env.redis.conn
}

func (env *enviroment) init() error {
	env.unsafe = config.Unsafe
	if config.Redis != "" {
		if err := cmdConnect.run(context.Background(), env, []string{config.Redis}); err != nil {
			return err
		}
	}
	return nil
}

func (env *enviroment) ps1() string {
	if env.more.size > 0 {
		return fmt.Sprintf("Display all %d %s? (y or n)", env.more.size, env.more.what)
	}
	var redis string
	if env.redisc() != nil {
		redis = "(redis=" + env.redisc().Options().Addr + ")"
	}
	return fmt.Sprintf("%s> ", redis)
}

func (env *enviroment) moreOut(size int, what string) io.Writer {
	f, ok := env.stdout.(*os.File)
	if !ok {
		return env.stdout
	}
	fd := f.Fd()
	if isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd) {
		env.more.size = size
		env.more.what = what
		env.more.content.Reset()
		return &env.more.content
	}
	return env.stdout
}

func (env *enviroment) shutdown() {
	if c := env.redisc(); c != nil {
		c.Close()
	}
	if env.logfile != nil {
		env.logfile.Close()
	}
}

func (env *enviroment) uiEnabled() bool {
	return !env.forked
}

func (env *enviroment) initUI() bool {
	if !env.uiEnabled() {
		return false
	}
	return termui.Init() == nil
}

func (env *enviroment) newFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet(env.name, flag.ContinueOnError)
	flagSet.SetOutput(env.stdout)
	return flagSet
}

func (env *enviroment) doFmt(format string, a ...interface{}) string {
	if len(a) == 0 {
		return format
	}
	return fmt.Sprintf(format, a...)
}

func (env *enviroment) errorf(format string, a ...interface{}) {
	msg := "(error) " + env.doFmt(format, a...)
	env.wmu.Lock()
	defer env.wmu.Unlock()
	fmt.Fprint(env.stderr, msg)
}

func (env *enviroment) errorln(format string, a ...interface{}) {
	msg := "(error) " + env.doFmt(format, a...)
	env.wmu.Lock()
	defer env.wmu.Unlock()
	if msg[len(msg)-1] == '\n' {
		fmt.Fprint(env.stderr, msg)
	} else {
		fmt.Fprintln(env.stderr, msg)
	}
}

func (env *enviroment) printf(format string, a ...interface{}) {
	env.fprintf(env.stdout, format, a...)
}

func (env *enviroment) fprintf(w io.Writer, format string, a ...interface{}) {
	msg := env.doFmt(format, a...)
	env.wmu.Lock()
	defer env.wmu.Unlock()
	fmt.Fprint(w, msg)
}

func (env *enviroment) println(format string, a ...interface{}) {
	env.fprintln(env.stdout, format, a...)
}

func (env *enviroment) fprintln(w io.Writer, format string, a ...interface{}) {
	msg := env.doFmt(format, a...)
	env.wmu.Lock()
	defer env.wmu.Unlock()
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		fmt.Fprint(w, msg)
	} else {
		fmt.Fprintln(w, msg)
	}
}

func (env *enviroment) appendStdin(args []string) []string {
	if env.stdin == nil {
		return args
	}
	// append last argument stdin
	if stdin, err := io.ReadAll(env.stdin); err != nil {
		env.errorln("read stdin error: %v", err)
		return args
	} else if len(stdin) > 0 {
		if stdin[len(stdin)-1] == '\n' {
			args = append(args, string(stdin[:len(stdin)-1]))
		} else {
			args = append(args, string(stdin))
		}
	}
	return args
}

func (env *enviroment) run(ctx context.Context, line string, stdin io.Reader, stdout io.Writer) {
	env.stdout = stdout
	args, err := shell.Split(line)
	if err != nil {
		env.errorln(err.Error())
		return
	}
	if len(args) == 0 {
		return
	}
	var (
		last int
		cmds [][]string
	)
	// split by pipe char '|'
	for i := 0; i < len(args); i++ {
		if args[i] == "|" {
			cmds = append(cmds, args[last:i])
			last = i + 1
			if i+1 == len(args) {
				env.errorln("missing command after last |")
				return
			}
			continue
		}
		if i+1 == len(args) {
			cmds = append(cmds, args[last:i+1])
		}
	}

	var out *bytes.Buffer
	n := len(cmds)
	for i := range cmds {
		if out == nil {
			env.stdin = stdin
		} else {
			env.stdin = out
		}
		if i == n-1 {
			env.stdout = stdout
		} else {
			out = bytes.NewBuffer(nil)
			env.stdout = out
		}
		if err := env.exec(ctx, cmds[i]); err != nil {
			if _, ee, ok := toExitError(err); ok {
				env.exitCode = ee.code
				if ee.reason != "" {
					env.errorln("(%d) %s", ee.code, ee.reason)
				}
			} else {
				env.exitCode = 2
			}
		}
	}
}

func (env *enviroment) exec(ctx context.Context, args []string) error {
	env.exitCode = 0
	var (
		name    = args[0]
		off     = 1
		cmd, ok = commands[name]
	)
	env.name = name
	if !ok {
		isRedisCommand, isShellCommand := env.find(name)
		if isRedisCommand && !isShellCommand {
			cmd = cmdRedis
			off = 0
		} else if !isRedisCommand && isShellCommand {
			cmd = cmdShell
			off = 0
		} else if isRedisCommand && isShellCommand {
			env.errorln("command %q ambigous, preappend sh (or redis) to run shell (or redis) command.", name)
			env.errorln("examples:")
			env.errorln("\tsh %s ...", name)
			env.errorln("\tredis %s ...", name)
			return nil
		} else {
			env.errorln("command %q not found", name)
			return nil
		}
	}

	if cmd.appendStdin {
		args = env.appendStdin(args)
	}
	return cmd.run(ctx, env, args[off:])
}

func (env *enviroment) find(name string) (isRedisCommand, isShellCommand bool) {
	if env.redis.commands != nil {
		if _, ok := env.redis.commands[strings.ToLower(name)]; ok {
			isRedisCommand = true
		}
	}
	isShellCommand = lookupShellCommand(name)
	return
}
