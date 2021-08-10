package main

import "strings"

type completion struct {
	commands []string
}

func (c *completion) add(commandName string) {
	for i := range c.commands {
		if c.commands[i] == commandName {
			return
		}
	}
	c.commands = append(c.commands, commandName)
}

func (c *completion) search(prefix string, limit int) []string {
	var result []string
	for i := range c.commands {
		if strings.HasPrefix(c.commands[i], prefix) {
			result = append(result, c.commands[i])
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result
}
