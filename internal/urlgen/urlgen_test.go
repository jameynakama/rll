package urlgen_test

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/jameynakama/reallylonglink/internal/urlgen"
)

var (
	validSegmentBase = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	validExtension   = regexp.MustCompile(`\.(php|aspx|html)$`)
	hyphenatedWords  = regexp.MustCompile(`^[a-z]+(-[a-z]+)+$`)
	numericSegment   = regexp.MustCompile(`^\d+$`)
	alphanumericID   = regexp.MustCompile(`^[a-z0-9]{6,12}$`)
	queryKeyPattern  = regexp.MustCompile(`^[a-z][a-z_-]*$`)
	queryValueRe     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

func parse(t *testing.T, result string) *url.URL {
	t.Helper()
	u, err := url.Parse("https://dogs.com/" + result)
	if err != nil {
		t.Fatalf("Generate() produced unparseable output: %v", err)
	}
	return u
}

func pathSegments(t *testing.T, result string) []string {
	t.Helper()
	u := parse(t, result)
	return strings.Split(strings.Trim(u.Path, "/"), "/")
}

// Should have at least two path segments
func TestGenerateHasMultiplePathSegments(t *testing.T) {
	parts := pathSegments(t, urlgen.Generate())
	if len(parts) < 2 {
		t.Errorf("expected at least 2 path segments; got %d", len(parts))
	}
}

// Should have path segments containing only safe URL characters
func TestGeneratePathSegmentsAreURLSafe(t *testing.T) {
	for _, seg := range pathSegments(t, urlgen.Generate()) {
		base := validExtension.ReplaceAllString(seg, "")
		if !validSegmentBase.MatchString(base) {
			t.Errorf("segment %q contains unsafe characters", seg)
		}
	}
}

// Should have at least two query params
func TestGenerateHasMultipleQueryParams(t *testing.T) {
	u := parse(t, urlgen.Generate())
	if len(u.Query()) < 2 {
		t.Errorf("expected at least 2 query params; got %d", len(u.Query()))
	}
}

// Should have query keys that are lowercase letters and underscores only
func TestGenerateQueryKeysAreLowercaseWithUnderscores(t *testing.T) {
	u := parse(t, urlgen.Generate())
	for k := range u.Query() {
		if !queryKeyPattern.MatchString(k) {
			t.Errorf("query key %q should be lowercase letters/underscores only", k)
		}
	}
}

// Should have query values that are lowercase alphanumeric or hyphenated only
func TestGenerateQueryValuesAreLowercaseAlphanumeric(t *testing.T) {
	u := parse(t, urlgen.Generate())
	for _, vals := range u.Query() {
		for _, v := range vals {
			if !queryValueRe.MatchString(v) {
				t.Errorf("query value %q contains unsafe characters", v)
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

// Should occasionally produce numeric path segments like /48291/
func TestGenerateOccasionallyHasNumericSegment(t *testing.T) {
	for range 100 {
		for _, seg := range pathSegments(t, urlgen.Generate()) {
			if numericSegment.MatchString(seg) {
				return
			}
		}
	}
	t.Error("expected a numeric segment in 100 calls; got none")
}

// Should occasionally produce alphanumeric ID segments like /a3f92b1c/
func TestGenerateOccasionallyHasAlphanumericIDSegment(t *testing.T) {
	for range 100 {
		for _, seg := range pathSegments(t, urlgen.Generate()) {
			if alphanumericID.MatchString(seg) && !hyphenatedWords.MatchString(seg) && !numericSegment.MatchString(seg) {
				return
			}
		}
	}
	t.Error("expected an alphanumeric ID segment in 100 calls; got none")
}

// Should occasionally produce path segments ending in .php, .aspx, or .html
func TestGenerateOccasionallyHasFileExtension(t *testing.T) {
	for range 100 {
		result := urlgen.Generate()
		path := strings.SplitN(result, "?", 2)[0]
		if validExtension.MatchString(path) {
			return
		}
	}
	t.Error("expected a file extension in 100 calls; got none")
}

// Should occasionally produce scammy query keys like utm_source, ref, token
func TestGenerateOccasionallyHasScammyQueryKey(t *testing.T) {
	for range 100 {
		u := parse(t, urlgen.Generate())
		for k := range u.Query() {
			if strings.Contains(k, "_") {
				return
			}
		}
	}
	t.Error("expected a scammy query key (with underscore) in 100 calls; got none")
}

// Should occasionally produce random alphanumeric query values containing digits
func TestGenerateOccasionallyHasRandomQueryValue(t *testing.T) {
	for range 100 {
		u := parse(t, urlgen.Generate())
		for _, vals := range u.Query() {
			for _, v := range vals {
				if regexp.MustCompile(`[0-9]`).MatchString(v) {
					return
				}
			}
		}
	}
	t.Error("expected a random alphanumeric query value (containing digit) in 100 calls; got none")
}
