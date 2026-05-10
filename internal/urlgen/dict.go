package urlgen

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"strings"
)

const dictPath = "/usr/share/dict/words"

var words []string

func init() {
	f, err := os.Open(dictPath)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		word := s.Text()
		if strings.IndexFunc(word, func(r rune) bool {
			return r < 'a' || r > 'z'
		}) == -1 {
			words = append(words, word)
		}
	}
}

func randomWord() string {
	return words[rand.Intn(len(words))]
}
