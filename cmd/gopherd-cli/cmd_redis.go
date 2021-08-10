package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

//go:embed redis.txt
var redisTxt []byte

type redisCommandInfo struct {
	syntax string
	usage  string
}

var redisCompletions = make(map[string]*redisCommandInfo)

func init() {
	s := bufio.NewScanner(bytes.NewReader(redisTxt))
	for s.Scan() {
		var syntax = s.Text()
		if strings.HasPrefix(syntax, "#") {
			continue
		}
		var usage string
		for {
			if !s.Scan() {
				return
			}
			usage = s.Text()
			if !strings.HasPrefix(usage, "#") {
				break
			}
		}
		if i := strings.IndexByte(syntax, ' '); i > 0 {
			redisCompletions[strings.ToLower(syntax[:i])] = &redisCommandInfo{
				syntax: syntax[i+1:],
				usage:  usage,
			}
		} else {
			redisCompletions[strings.ToLower(syntax)] = &redisCommandInfo{
				usage: usage,
			}
		}
	}
}

var cmdConnect = register(&command{
	name:   "connect-redis",
	format: "[host:port]",
	usage:  "connect to redis server",
	run: func(ctx context.Context, env *enviroment, args []string) error {
		var addr = "127.0.0.1:6379"
		if len(args) > 0 {
			addr = args[0]
			if i := strings.Index(addr, ":"); i == 0 {
				addr = "127.0.0.1" + addr
			} else if i < 0 {
				addr += ":6379"
			}
		}
		client := redis.NewClient(&redis.Options{
			Addr: addr,
		})
		if cmds, err := client.Command(context.Background()).Result(); err != nil {
			env.errorln(err.Error())
			return err
		} else {
			env.redis.commands = make(map[string]*redis.CommandInfo)
			for _, cmd := range cmds {
				if !cmd.ReadOnly {
					readonly := true
					for i := range cmd.Flags {
						if cmd.Flags[i] == "admin" || cmd.Flags[i] == "write" {
							readonly = false
						}
					}
					cmd.ReadOnly = readonly
				}
				complet.add(cmd.Name)
				env.redis.commands[cmd.Name] = cmd
			}
		}
		if c := env.redisc(); c != nil {
			c.Close()
		}
		env.redis.conn = client
		env.println("%s connected", addr)
		return nil
	},
})

var cmdRedis = register(&command{
	name:        "redis",
	format:      "<redis command>",
	usage:       "exec redis command with connected redis client",
	appendStdin: true,
	run: func(ctx context.Context, env *enviroment, args []string) error {
		if env.redisc() == nil {
			err := errors.New("redis client not found, please connect to redis server first!")
			env.errorln(err.Error())
			return err
		}
		cmd, ok := env.redis.commands[strings.ToLower(args[0])]
		if !ok {
			err := fmt.Errorf("redis command %q not supported", args[0])
			env.errorln(err.Error())
			return err
		}
		if !env.unsafe && !cmd.ReadOnly {
			env.errorln("redis command %q is not readonly.", args[0])
			env.errorln("run the tool with flag -unsafe to enable all redis commands.")
			return errors.New("permission denied")
		}
		iargs := make([]interface{}, 0, len(args))
		for _, arg := range args {
			iargs = append(iargs, arg)
		}
		res, err := env.redisc().Do(context.Background(), iargs...).Result()
		if err != nil {
			env.errorln(err.Error())
			return err
		}
		var (
			out   = env.stdout
			buf   bytes.Buffer
			lines = formatRESP(&buf, "", res)
		)
		if lines > 100 {
			out = env.moreOut(lines, "lines")
		}
		if buf.Len() > 0 {
			env.fprintln(out, buf.String())
		}
		return nil
	},
})

func formatRESP(w io.Writer, prefix string, x interface{}) int {
	lines := 0
	switch value := x.(type) {
	case []interface{}: // array
		maxnumber := len(value)
		if maxnumber > 0 {
			d := digits(maxnumber)
			const numbersuffix = ") "
			numberfmt := "%0" + strconv.Itoa(d) + "d"
			nextprefix := prefix + strings.Repeat(" ", d+len(numbersuffix))
			for i, v := range value {
				if i > 0 {
					fmt.Fprint(w, prefix)
				}
				fmt.Fprintf(w, numberfmt+numbersuffix, i+1)
				lines += formatRESP(w, nextprefix, v)
			}
		}
		return lines
	case string: // simple string
		fmt.Fprintf(w, "%q", value)
	case []byte: // bulk string
		fmt.Fprintf(w, "%q", value)
	case error: // error
		fmt.Fprintf(w, "(error) %v", value)
	default: // integer
		t := reflect.TypeOf(x)
		k := t.Kind()
		if k >= reflect.Int && k <= reflect.Uint64 {
			fmt.Fprintf(w, "(integer) %v", value)
		} else {
			fmt.Fprintf(w, "%v", value)
		}
	}
	w.Write([]byte("\n"))
	lines++
	return lines
}
