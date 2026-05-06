# Web Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a server-rendered HTML frontend (two pages: home form + result) to the Go server, replicating the existing Django site at reallylong.link.

**Architecture:** PRG pattern -- GET / renders a form, POST / creates a link and redirects to GET /links/{id} which renders the result. Templates are embedded in the binary via `//go:embed`. Web handlers live alongside the JSON handlers in `internal/api/`, sharing the same `Handler` struct and query layer.

**Tech Stack:** Go `html/template`, `embed`, chi router (already in use), standard `httptest` for tests.

---

### Task 1: Extract `generateReallyLongUrl` helper

**Files:**
- Modify: `internal/api/handlers.go`

The link-generation logic currently lives inline in `createLink`. The new `webCreateLink` handler needs the same logic. Extract it to a package-level function to avoid duplication.

- [ ] **Step 1: Extract the function**

In `internal/api/handlers.go`, add this function above `createLink`, and replace the inline byte-slice generation in `createLink` with a call to it.

Add after the `const` declarations:
```go
func generateReallyLongUrl() string {
	b := make([]byte, maxURLLength)
	for i := range b {
		b[i] = availableChars[rand.Intn(len(availableChars))]
	}
	return string(b)
}
```

Replace the inline generation block inside `createLink` (the `reallyLongUrl := make([]byte, maxURLLength)` through `string(reallyLongUrl)`) with:
```go
	row, err := h.queries.CreateLink(r.Context(), store.CreateLinkParams{
		OriginalUrl:   req.OriginalUrl,
		ReallyLongUrl: generateReallyLongUrl(),
	})
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```
Expected: no output, exit 0.

- [ ] **Step 3: Run existing tests to confirm no regression**

```bash
just test -run TestCreateLink -v
```
Expected: `PASS`

- [ ] **Step 4: Commit**

```bash
jj describe -m "Extract generateReallyLongUrl helper" && jj new
```

---

### Task 2: Create HTML templates

**Files:**
- Create: `internal/api/templates/index.html`
- Create: `internal/api/templates/result.html`

- [ ] **Step 1: Create `internal/api/templates/index.html`**

```html
<!DOCTYPE html>
<html>
<body>
<h1>Really Long Link</h1>
<p><b>Link extension for the everyday Net Surfer</b></p>
{{if .Error}}<p>{{.Error}}</p>{{end}}
<form method="post" action="/">
Original link: <input type="text" name="original_url"> <input type="submit" value="Make it really really long, please">
</form>
</body>
</html>
```

- [ ] **Step 2: Create `internal/api/templates/result.html`**

```html
<!DOCTYPE html>
<html>
<body>
<p><a href="/">Home</a></p>
<p>original: <a href="{{.OriginalUrl}}">{{.OriginalUrl}}</a></p>
<p>really long link:<br>
<a href="{{.RedirectURL}}">{{.RedirectURL}}</a></p>
<textarea rows="10" cols="80">{{.RedirectURL}}</textarea>
</body>
</html>
```

- [ ] **Step 3: Commit**

```bash
jj describe -m "Add HTML templates for web frontend" && jj new
```

---

### Task 3: Stub web.go + wire routes

**Files:**
- Create: `internal/api/web.go`
- Modify: `internal/api/router.go`

Create handler stubs that return 501 so tests can compile and fail meaningfully. Wire the routes.

- [ ] **Step 1: Create `internal/api/web.go`**

```go
package api

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates
var templateFiles embed.FS

var (
	indexTmpl  = template.Must(template.ParseFS(templateFiles, "templates/index.html"))
	resultTmpl = template.Must(template.ParseFS(templateFiles, "templates/result.html"))
)

type indexData struct {
	Error string
}

type resultData struct {
	OriginalUrl string
	RedirectURL string
}

func (h *Handler) webIndex(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) webCreateLink(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) webGetLink(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
```

- [ ] **Step 2: Add routes to `internal/api/router.go`**

Add these three lines inside `NewRouter`, before the `r.Route("/api/v1", ...)` block:

```go
	r.Get("/", h.webIndex)
	r.Post("/", h.webCreateLink)
	r.Get("/links/{id}", h.webGetLink)
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```
Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
jj describe -m "Add web handler stubs and routes" && jj new
```

---

### Task 4: Implement and test webIndex

**Files:**
- Modify: `internal/api/handlers_test.go`
- Modify: `internal/api/web.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/api/handlers_test.go`:

```go
func TestWebIndex(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Really Long Link") {
		t.Errorf("expected 'Really Long Link' in body; got %s", w.Body.String())
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
just test -run TestWebIndex -v
```
Expected: `FAIL` -- `expected 200; got 501`

- [ ] **Step 3: Implement `webIndex` in `internal/api/web.go`**

Replace the stub:

```go
func (h *Handler) webIndex(w http.ResponseWriter, r *http.Request) {
	if err := indexTmpl.Execute(w, indexData{}); err != nil {
		log.Printf("webIndex: %v", err)
	}
}
```

Add `"log"` to the import block in `web.go`.

- [ ] **Step 4: Run the test to verify it passes**

```bash
just test -run TestWebIndex -v
```
Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
jj describe -m "Implement webIndex" && jj new
```

---

### Task 5: Implement and test webCreateLink

**Files:**
- Modify: `internal/api/handlers_test.go`
- Modify: `internal/api/web.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/api/handlers_test.go`:

```go
func TestWebCreateLink(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	body := strings.NewReader("original_url=https%3A%2F%2Fexample.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303; got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Result().Header.Get("Location")
	if !strings.HasPrefix(loc, "/links/") {
		t.Errorf("expected redirect to /links/{id}; got %s", loc)
	}
}

func TestWebCreateLinkEmptyURL(t *testing.T) {
	srv := newTestServer(t)
	body := strings.NewReader("original_url=")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "original url is required") {
		t.Errorf("expected error message in body; got %s", w.Body.String())
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
just test -run "TestWebCreateLink" -v
```
Expected: both `FAIL` -- `expected 303; got 501` and `expected 422; got 501`

- [ ] **Step 3: Implement `webCreateLink` in `internal/api/web.go`**

Replace the stub. Also add `"fmt"` and `"github.com/jameynakama/reallylonglink/internal/store"` to the import block.

```go
func (h *Handler) webCreateLink(w http.ResponseWriter, r *http.Request) {
	originalUrl := r.FormValue("original_url")
	if originalUrl == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := indexTmpl.Execute(w, indexData{Error: "original url is required"}); err != nil {
			log.Printf("webCreateLink: render: %v", err)
		}
		return
	}

	row, err := h.queries.CreateLink(r.Context(), store.CreateLinkParams{
		OriginalUrl:   originalUrl,
		ReallyLongUrl: generateReallyLongUrl(),
	})
	if err != nil {
		log.Printf("webCreateLink: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/links/%d", row.ID), http.StatusSeeOther)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
just test -run "TestWebCreateLink" -v
```
Expected: both `PASS`

- [ ] **Step 5: Commit**

```bash
jj describe -m "Implement webCreateLink" && jj new
```

---

### Task 6: Implement and test webGetLink

**Files:**
- Modify: `internal/api/handlers_test.go`
- Modify: `internal/api/web.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/api/handlers_test.go`:

```go
func TestWebGetLink(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl")
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/links/%d", link.ID), nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), link.OriginalUrl) {
		t.Errorf("expected original url in body; got %s", w.Body.String())
	}
}

func TestWebGetLink404(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/links/999", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
just test -run "TestWebGetLink" -v
```
Expected: both `FAIL` -- `expected 200; got 501` and `expected 404; got 501`

- [ ] **Step 3: Implement `webGetLink` in `internal/api/web.go`**

Replace the stub. Also add `"errors"`, `"strconv"`, `"github.com/go-chi/chi/v5"`, and `"github.com/jackc/pgx/v5"` to the import block.

```go
func (h *Handler) webGetLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	row, err := h.queries.GetLink(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("webGetLink: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	redirectURL := fmt.Sprintf("%s://%s/api/v1/rll/%s", scheme, r.Host, row.ReallyLongUrl)

	if err := resultTmpl.Execute(w, resultData{
		OriginalUrl: row.OriginalUrl,
		RedirectURL: redirectURL,
	}); err != nil {
		log.Printf("webGetLink: render: %v", err)
	}
}
```

- [ ] **Step 4: Run all tests to verify everything passes**

```bash
just test -v
```
Expected: all `PASS`

- [ ] **Step 5: Commit**

```bash
jj describe -m "Implement webGetLink" && jj new
```
