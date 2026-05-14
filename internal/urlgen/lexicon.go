package urlgen

import (
	_ "embed"
	"fmt"
	"math/rand"
	"strings"
)

const (
	nounTag = "N"
	advTag  = "v"
	adjTag  = "A"
)

//go:embed moby.pos
var mobyData string

var adjectives []string
var adverbs []string
var nouns []string

var wordsByPOS map[string][]string

func init() {
	buildPOSLists()
}

func buildPOSLists() {
	words := strings.SplitSeq(mobyData, "\n")
	for line := range words {
		dat := strings.Split(line, "\\")
		if len(dat) != 2 {
			continue
		}
		w := dat[0]
		pos := dat[1]
		if strings.ContainsFunc(w, func(r rune) bool {
			return r < 'a' || r > 'z'
		}) {
			continue
		}
		if strings.Contains(pos, nounTag) {
			nouns = append(nouns, w)
		}
		if strings.Contains(pos, advTag) {
			adverbs = append(adverbs, w)
		}
		if strings.Contains(pos, adjTag) {
			adjectives = append(adjectives, w)
		}
	}
	wordsByPOS = map[string][]string{
		"adjective": adjectives,
		"adverb":    adverbs,
		"noun":      nouns,
	}
}

func randomWord(pos string) string {
	words := wordsByPOS[pos]
	return words[rand.Intn(len(words))]
}

func randomPhrase(length int) string {
	switch length {
	case 2:
		return fmt.Sprintf("%s-%s", randomWord("adjective"), randomWord("noun"))
	case 3:
		return fmt.Sprintf("%s-%s-%s", randomWord("adverb"), randomWord("adjective"), randomWord("noun"))
	default:
		return randomWord("noun")
	}
}
