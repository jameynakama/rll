package urlgen

import (
	"math/rand"
	"strconv"
	"strings"
)

const (
	minPaths         = 100
	maxPaths         = 120
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

const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomAlphanumericID() string {
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
		fallthrough
	case 2:
		return randomAlphanumericID()
	case 3:
		fallthrough
	default:
		return randomPhrase(rand.Intn(3) + 1)
	}
}

func randomQueryKey() string {
	if rand.Intn(2) == 0 {
		return scammyKeys[rand.Intn(len(scammyKeys))]
	}
	return randomPhrase(rand.Intn(maxQueryArgWords-minQueryArgWords+1) + minQueryArgWords)
}

func randomQueryValue() string {
	if rand.Intn(2) == 0 {
		return randomAlphanumericID()
	}
	return randomPhrase(rand.Intn(maxQueryArgWords-minQueryArgWords+1) + minQueryArgWords)
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
