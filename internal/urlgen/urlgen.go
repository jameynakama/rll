package urlgen

import (
	"math/rand"
	"strconv"
	"strings"
)

const (
	minPaths         = 50
	maxPaths         = 70
	minPathWords     = 2
	maxPathWords     = 4
	minQueryArgs     = 12
	maxQueryArgs     = 20
	minQueryArgWords = 1
	maxQueryArgWords = 2
)

var fileExts = []string{".php", ".aspx", ".html"}

var scammyKeys = []string{
	"utm_source", "utm_medium", "utm_campaign", "ref", "sessionid",
	"token", "id", "user_id", "tracking", "clickid", "fbclid", "gclid",
}

func randomAlphanumericID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, rand.Intn(7)+6) // 6-12 chars
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func randomPathSegment() string {
	switch rand.Intn(5) {
	case 0:
		return strconv.Itoa(rand.Intn(9000000) + 100000) // 6-7 digit number
	case 1:
		return randomAlphanumericID()
	default:
		parts := make([]string, rand.Intn(maxPathWords-minPathWords+1)+minPathWords)
		for i := range parts {
			parts[i] = randomWord()
		}
		return strings.Join(parts, "-")
	}
}

func randomQueryKey() string {
	if rand.Intn(2) == 0 {
		return scammyKeys[rand.Intn(len(scammyKeys))]
	}
	var parts []string
	for range rand.Intn(maxQueryArgWords-minQueryArgWords+1) + minQueryArgWords {
		parts = append(parts, randomWord())
	}
	return strings.Join(parts, "-")
}

func randomQueryValue() string {
	if rand.Intn(3) == 0 {
		return randomAlphanumericID()
	}
	var parts []string
	for range rand.Intn(maxQueryArgWords-minQueryArgWords+1) + minQueryArgWords {
		parts = append(parts, randomWord())
	}
	return strings.Join(parts, "-")
}

func Generate() (string, string) {
	segs := make([]string, rand.Intn(maxPaths-minPaths+1)+minPaths) // [minPaths, maxPaths] inclusive
	for i := range segs {
		segs[i] = randomPathSegment()
		if i == len(segs)-1 && rand.Intn(4) == 0 {
			segs[i] += fileExts[rand.Intn(len(fileExts))]
		}
	}
	path := strings.Join(segs, "/")

	q := make([]string, rand.Intn(maxQueryArgs-minQueryArgs+1)+minQueryArgs) // [minQueryArgs, maxQueryArgs] inclusive
	for i := range q {
		q[i] = randomQueryKey() + "=" + randomQueryValue()
	}
	query := "?" + strings.Join(q, "&")

	return path, query
}
