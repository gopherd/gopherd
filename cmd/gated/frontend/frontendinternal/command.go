package frontendinternal

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gopherd/doge/proto"
)

type command struct {
	name   string
	format string
	usage  string
	run    func(*frontendComponent, *session, []string) error
}

var (
	commands = make(map[string]*command)
)

func registerCommand(cmd *command) {
	if cmd.name == "" {
		panic("registerCommand a empty command")
	}
	if _, dup := commands[cmd.name]; dup {
		panic("registerCommand called twice for " + cmd.name)
	}
	commands[cmd.name] = cmd
}

func init() {
	// .help [command]
	registerCommand(&command{
		name:   "help",
		format: "[command]",
		usage:  "show help information",
		run: func(f *frontendComponent, sess *session, args []string) error {
			var cmds []*command
			if len(args) > 0 {
				for i := range args {
					cmd := commands[strings.ToLower(args[i])]
					if cmd == nil {
						return sess.println("command", args[i], "not found")
					}
					cmds = append(cmds, cmd)
				}
			} else {
				for _, cmd := range commands {
					cmds = append(cmds, cmd)
				}
				sort.Slice(cmds, func(i, j int) bool {
					return cmds[i].name < cmds[j].name
				})
			}
			for _, cmd := range cmds {
				if cmd.format != "" {
					if err := sess.println("."+cmd.name+" ", cmd.format); err != nil {
						return err
					}
				} else {
					if err := sess.println("." + cmd.name); err != nil {
						return err
					}
				}
				if err := sess.println("\t" + cmd.usage); err != nil {
					return err
				}
			}
			return nil
		},
	})

	// .echo [content]
	registerCommand(&command{
		name:   "echo",
		format: "[content]",
		usage:  "echo content",
		run: func(f *frontendComponent, sess *session, args []string) error {
			for i := range args {
				if i > 0 {
					if err := sess.print(" "); err != nil {
						return err
					}
				}
				if err := sess.print(args[i]); err != nil {
					return err
				}
			}
			return sess.println()
		},
	})

	// .send <type> [json]
	registerCommand(&command{
		name:   "send",
		format: "<type> [json]",
		usage:  "send message by type with json formatted content",
		run: func(f *frontendComponent, sess *session, args []string) error {
			if len(args) < 1 {
				return sess.println("argument <type> required")
			}
			typ, err := strconv.Atoi(args[0])
			if err != nil {
				return sess.println("argument <type> invalid")
			}
			if typ <= 0 || typ > proto.MaxType {
				return sess.println("argument <type> out of range")
			}
			body := proto.Text([]byte(strings.Join(args[1:], "")))
			return f.onMessage(sess, proto.Type(typ), body)
		},
	})
}
