# Web Frontend Design

**Date:** 2026-05-06
**Status:** Approved

## Summary

Add a server-rendered HTML frontend to replace the existing Django app at reallylong.link. Two pages: a home page with the link-creation form, and a result page showing the original and generated URLs. No CSS, no JS -- faithful to the existing site's aesthetic.

## Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/` | `webIndex` | Render home page with form |
| POST | `/` | `webCreateLink` | Create link, redirect to result |
| GET | `/links/{id}` | `webGetLink` | Render result page |

Uses Post/Redirect/Get pattern. Browser back button and refresh are safe on the result page.

## File Structure

```
internal/api/
  web.go                  # template handlers + go:embed
  templates/
    index.html            # home page
    result.html           # result page
```

Templates are embedded into the binary via `//go:embed`.

## Templates

### index.html

- `<h1>Really Long Link</h1>`
- `<p><b>Link extension for the everyday Net Surfer</b></p>`
- Form: `method="post" action="/"`, inline label + text input + submit button ("Make it really really long, please")
- If validation error: plain inline message above the form

### result.html

- `<a href="/">Home</a>`
- `original: <a href="...">...</a>`
- `really long link:` label, then the full redirect URL as a link
- `<textarea>` pre-filled with the full redirect URL for easy copying

The redirect URL shown to the user is the full URL including scheme and host, e.g. `https://reallylong.link/api/v1/rll/{really_long_url}`. It is constructed in the handler from the incoming request (`r.Host`, `X-Forwarded-Proto` if present, else `http`).

## Error Handling

- Empty `original_url` on POST: re-render `index.html` with a message above the form (HTTP 422)
- DB or server error: plain-text 500 response
- Unknown ID on result page: plain-text 404

## Testing

- Tests live in `internal/api/handlers_test.go` alongside existing tests
- Same pattern: `httptest.NewRequest` + real DB via `testPool`
- Cover: home page renders (200), form submit creates link and redirects (303), result page renders correct data (200), empty URL returns 422, unknown ID returns 404
