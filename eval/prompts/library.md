Create a small library package for a URL shortener. The library should provide:
- A `Shortener` type that maps short codes to full URLs
- `Shorten(url string) (code string, err error)` — generates a random 6-character alphanumeric code
- `Resolve(code string) (url string, err error)` — returns the original URL or an error if not found
- `Load(path string) error` — loads mappings from a JSON file
- `Save(path string) error` — saves mappings to a JSON file
- Thread-safe concurrent access

The project should be written in Go, compile with `go build`, and include a small example program in `main.go` that demonstrates the library.
