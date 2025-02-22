package main

/**

15/02/2025

(1) Can we add a filter langauge option to the table so the api fetches that content for that language (we can start with 10 languages)
(2) Let's highlight the row that's being shown on the right
(3) Let's add some syntax highlighting to the viewer
(4) let's add caching on the serverside for each paste so we don't need to refetch it
(5) Let's add a copy functionality to the viewer

I'm happy to call the project a success as a v1.
---------------------------------
15/02/2025

For v2 here's the list
(1) Let's update the rows to include the username
(2) pastebin provies a way to view all pastes created by a user, so clicking the username will show us all his pastes
(3) Another button in the viewer to clone a paste (i.e copy the contents of the paste and create it in my account using the api)
---------------------------------
16/02/2025

There was a problem with pastebin's api,
I'm getting 403 for any api including login. Have messaged customer support.

Pivoting to a different v2.

---------------------------------
16/02/2025

Let's pivot to v2 in a different way. There are some programs that I think are safe to run locally. Let's run those on the browser in required language with support for some standard libraries.
The new layout contains three parts, the viewer splits into two. A viewer from earlier and a cli window below it. Where we can run and interact with the program.
*/

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type Paste struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	Size      string `json:"size"`
	Syntax    string `json:"syntax"`
	ScrapeURL string `json:"scrape_url"`
	User      string `json:"user"`
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type CachedPaste struct {
	Content   string
	CachedAt  time.Time
	ExpiresAt time.Time
}

var (
	pasteCache = make(map[string]CachedPaste)
	cacheMutex sync.RWMutex
)

type RunRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type RunResponse struct {
	Output string `json:"output"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic("Error loading .env file")
	}

	e := echo.New()

	// Serve static files
	e.Static("/static", "static")

	t := &Template{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
	e.Renderer = t

	// Routes
	e.GET("/", handleHome)
	e.GET("/pastes", handleListPastes)
	e.GET("/paste/:key", handleViewPaste)
	e.POST("/run", handleRun)

	e.Start(":8080")
}

func handleHome(c echo.Context) error {
	return c.Render(http.StatusOK, "home.html", nil)
}

func handleListPastes(c echo.Context) error {
	language := c.QueryParam("language")
	url := "https://scrape.pastebin.com/api_scraping.php?limit=250"

	if language != "" && language != "all" {
		url += "&lang=" + language
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching: %v\n", err)
		return c.HTML(http.StatusInternalServerError, "Error fetching pastes")
	}
	defer resp.Body.Close()

	var pastes []Paste
	if err := json.NewDecoder(resp.Body).Decode(&pastes); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return c.HTML(http.StatusInternalServerError, "Error parsing pastes: "+err.Error())
	}

	if len(pastes) == 0 {
		return c.HTML(http.StatusOK, `
            <div class="text-gray-500 text-center py-4">
                No pastes found
            </div>
        `)
	}

	return c.HTML(http.StatusOK, renderPasteList(pastes))
}

func renderPasteList(pastes []Paste) string {
	html := `<div class="divide-y">`
	for _, paste := range pastes {
		title := paste.Title
		if title == "" {
			title = "Untitled"
		}

		author := paste.User
		if author == "" {
			author = "Anonymous"
		}

		timestamp, _ := strconv.ParseInt(paste.Date, 10, 64)
		createdAt := time.Unix(timestamp, 0)

		html += fmt.Sprintf(`
            <div class="py-2 px-2 cursor-pointer paste-row"
                 onclick="document.querySelectorAll('.paste-row').forEach(el => el.classList.remove('bg-blue-100')); this.classList.add('bg-blue-100')"
                 hx-get="/paste/%s?language=%s"
                 hx-target="#paste-content"
                 hx-trigger="click">
                <div class="font-medium">%s</div>
                <div class="text-sm text-gray-500 flex items-center justify-between">
                    <div>
                        <span class="text-blue-600">@%s</span>
                        <span class="ml-2">%s</span>
                    </div>
                    <div>
                        <span class="px-2 py-1 bg-gray-100 rounded-full text-xs">
                            %s
                        </span>
                        <span class="ml-2 text-xs">
                            %s bytes
                        </span>
                    </div>
                </div>
            </div>`,
			paste.Key, paste.Syntax,
			title,
			author,
			createdAt.Format("2006-01-02 15:04"),
			paste.Syntax,
			paste.Size,
		)
	}
	html += `</div>`
	return html
}

func handleViewPaste(c echo.Context) error {
	key := c.Param("key")
	language := c.QueryParam("language")

	// Map Pastebin syntax to supported languages
	languageMap := map[string]string{
		"go":         "go",
		"python":     "python",
		"ruby":       "ruby",
		"javascript": "javascript",
		"java":       "java",
		"cpp":        "cpp",
		"csharp":     "csharp",
		"php":        "php",
		"swift":      "swift",
		"rust":       "rust",
	}

	if mapped, ok := languageMap[language]; ok {
		language = mapped
	} else {
		language = "plaintext"
	}

	const copyScript = `
		const toast = document.getElementById('toast');
		navigator.clipboard.writeText(this.nextElementSibling.textContent);
		toast.classList.add('show');
		setTimeout(() => toast.classList.remove('show'), 2000);
	`

	// Check cache first
	cacheMutex.RLock()
	if cached, exists := pasteCache[key]; exists && time.Now().Before(cached.ExpiresAt) {
		cacheMutex.RUnlock()
		return c.HTML(http.StatusOK, fmt.Sprintf(`
            <div class="relative h-full overflow-y-auto">
                <button onclick="%s" 
                        class="absolute top-2 right-2 px-2 py-1 bg-gray-200 hover:bg-gray-300 rounded text-sm">
                    Copy
                </button>
                <pre class="bg-gray-50 rounded p-2 h-full"><code class="language-%s">%s</code></pre>
            </div>
        `, copyScript, language, cached.Content))
	}
	cacheMutex.RUnlock()

	// If not in cache, fetch from API
	url := fmt.Sprintf("https://scrape.pastebin.com/api_scrape_item.php?i=%s", key)
	resp, err := http.Get(url)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, "Error fetching paste content")
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, "Error reading paste content")
	}

	// Store in cache
	cacheMutex.Lock()
	pasteCache[key] = CachedPaste{
		Content:   string(content),
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	cacheMutex.Unlock()

	return c.HTML(http.StatusOK, fmt.Sprintf(`
        <div class="relative h-full overflow-y-auto">
            <button onclick="%s" 
                    class="absolute top-2 right-2 px-2 py-1 bg-gray-200 hover:bg-gray-300 rounded text-sm">
                Copy
            </button>
            <pre class="bg-gray-50 rounded p-2 h-full"><code class="language-%s">%s</code></pre>
        </div>
    `, copyScript, language, content))
}

func handleRun(c echo.Context) error {
	var req RunRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, RunResponse{Output: "Invalid request"})
	}

	switch req.Language {
	case "go":
		return handleGoExecution(c, req)
	case "python":
		return handlePythonExecution(c, req)
	case "ruby":
		return handleRubyExecution(c, req)
	default:
		return c.JSON(http.StatusBadRequest, RunResponse{
			Output: "Unsupported language: " + req.Language,
		})
	}
}

func handleGoExecution(c echo.Context, req RunRequest) error {
	// Create a temporary directory
	dir, err := os.MkdirTemp("", "playground")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, RunResponse{
			Output: "Failed to create temp directory",
		})
	}
	defer os.RemoveAll(dir)

	// Write the code to a file
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(req.Code), 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, RunResponse{
			Output: "Failed to write code file",
		})
	}

	// Run the code
	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.JSON(http.StatusOK, RunResponse{
			Output: string(output),
		})
	}

	return c.JSON(http.StatusOK, RunResponse{
		Output: string(output),
	})
}

func handlePythonExecution(c echo.Context, req RunRequest) error {
	dir, err := os.MkdirTemp("", "playground")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, RunResponse{
			Output: "Failed to create temp directory",
		})
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte(req.Code), 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, RunResponse{
			Output: "Failed to write code file",
		})
	}

	cmd := exec.Command("python3", "main.py")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.JSON(http.StatusOK, RunResponse{
			Output: string(output),
		})
	}

	return c.JSON(http.StatusOK, RunResponse{
		Output: string(output),
	})
}

func handleRubyExecution(c echo.Context, req RunRequest) error {
	dir, err := os.MkdirTemp("", "playground")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, RunResponse{
			Output: "Failed to create temp directory",
		})
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "main.rb"), []byte(req.Code), 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, RunResponse{
			Output: "Failed to write code file",
		})
	}

	cmd := exec.Command("ruby", "main.rb")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.JSON(http.StatusOK, RunResponse{
			Output: string(output),
		})
	}

	return c.JSON(http.StatusOK, RunResponse{
		Output: string(output),
	})
}
