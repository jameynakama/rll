package urlgen_test

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/jameynakama/reallylonglink/internal/urlgen"
)

var segmentPattern = regexp.MustCompile(`^[a-z]+(-[a-z]+)+$`)
var queryKeyPattern = regexp.MustCompile(`^[a-z]+$`)
var queryValuePattern = regexp.MustCompile(`^[a-z0-9]+$`)

func parse(t *testing.T, result string) *url.URL {
	t.Helper()
	u, err := url.Parse("https://dogs.com" + result)
	if err != nil {
		t.Fatalf("Generate() produced unparseable output: %v", err)
	}
	return u
}

// Should have at least two path segments
func TestGenerateHasMultiplePathSegments(t *testing.T) {
	u := parse(t, urlgen.Generate())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		t.Errorf("expected at least 2 path segments; got %d: %q", len(parts), u.Path)
	}
}

// Should have path segments that look like hyphenated lowercase words
func TestGeneratePathSegmentsAreHyphenatedWords(t *testing.T) {
	u := parse(t, urlgen.Generate())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for _, seg := range parts {
		if !segmentPattern.MatchString(seg) {
			t.Errorf("segment %q should match hyphenated lowercase words (e.g. 'squirrely-monkey')", seg)
		}
	}
}

// Should have at least two query params
func TestGenerateHasMultipleQueryParams(t *testing.T) {
	u := parse(t, urlgen.Generate())
	if len(u.Query()) < 2 {
		t.Errorf("expected at least 2 query params; got %d in %q", len(u.Query()), u.RawQuery)
	}
}

// Should have query keys that are lowercase letters only
func TestGenerateQueryKeysAreLowercase(t *testing.T) {
	u := parse(t, urlgen.Generate())
	for k := range u.Query() {
		if !queryKeyPattern.MatchString(k) {
			t.Errorf("query key %q should be lowercase letters only", k)
		}
	}
}

// Should have query values that are lowercase alphanumeric only
func TestGenerateQueryValuesAreLowercaseAlphanumeric(t *testing.T) {
	u := parse(t, urlgen.Generate())
	for _, vals := range u.Query() {
		for _, v := range vals {
			if !queryValuePattern.MatchString(v) {
				t.Errorf("query value %q should be lowercase alphanumeric only", v)
			}
		}
	}
}

// Should produce different output each call
func TestGenerateIsRandom(t *testing.T) {
	seen := make(map[string]bool)
	for range 10 {
		seen[urlgen.Generate()] = true
	}
	if len(seen) < 5 {
		t.Errorf("Generate() not random enough: only %d unique results in 10 calls", len(seen))
	}
}
