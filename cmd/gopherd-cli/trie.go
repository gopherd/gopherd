package main

import "strings"

type trie struct {
	// TODO: using trie
	words []string
}

func (t *trie) Add(word string) {
	for i := range t.words {
		if t.words[i] == word {
			return
		}
	}
	t.words = append(t.words, word)
}

func (t *trie) Search(prefix string, limit int) []string {
	var result []string
	for i := range t.words {
		if strings.HasPrefix(t.words[i], prefix) {
			result = append(result, t.words[i])
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result
}
