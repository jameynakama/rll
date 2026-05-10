package urlgen

import (
	"math/rand"
	"strings"
)

const (
	minPaths         = 30
	maxPaths         = 50
	minPathWords     = 2
	maxPathWords     = 5
	minQueryArgs     = 5
	maxQueryArgs     = 10
	minQueryArgWords = 1
	maxQueryArgWords = 2
)

func Generate() string {
	var path strings.Builder

	for range rand.Intn(maxPaths-minPaths+1) + minPaths { // [minPaths, maxPaths] inclusive
		var words []string
		for range rand.Intn(maxPathWords-minPathWords+1) + minPathWords {
			words = append(words, randomWord())
		}
		path.WriteString("/" + strings.Join(words, "-"))
	}

	path.WriteString("?")
	var q []string
	for range rand.Intn(maxQueryArgs-minQueryArgs+1) + minQueryArgs {
		var k, v []string
		for range rand.Intn(maxQueryArgWords-minQueryArgWords+1) + minQueryArgWords {
			k = append(k, randomWord())
		}
		for range rand.Intn(maxQueryArgWords-minQueryArgWords+1) + minQueryArgWords {
			v = append(v, randomWord())
		}
		q = append(q, strings.Join(k, "-")+"="+strings.Join(v, "-"))
	}
	path.WriteString(strings.Join(q, "&"))

	return strings.TrimPrefix(path.String(), "/")
}
