package docs

import (
	"io/fs"
	"net/http"
	"strings"

	rootdocs "go-template/docs"

	"github.com/go-chi/chi/v5"
)

// Handler provides documentation endpoints
type Handler struct {
	docsFS fs.FS
}

// NewHandler creates a new documentation handler
func NewHandler() *Handler {
	return &Handler{
		docsFS: rootdocs.FS(),
	}
}

// Routes sets up the documentation routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Serve static documentation files
	r.Handle("/*", h.fileServer())

	// Specific routes for better UX
	r.Get("/", h.indexPage)
	r.Get("/redoc", h.redirectToRedocHTML)
	r.Get("/swagger-ui", h.redirectToSwaggerUI)

	return r
}

// fileServer serves embedded static files
func (h *Handler) fileServer() http.Handler {
	return http.StripPrefix("/docs/", http.FileServer(http.FS(h.docsFS)))
}

// indexPage serves a documentation index
func (h *Handler) indexPage(w http.ResponseWriter, r *http.Request) {
	indexHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Go Template API Documentation</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8f9fa;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            border-bottom: 2px solid #3498db;
            padding-bottom: 10px;
        }
        h2 {
            color: #34495e;
            margin-top: 30px;
        }
        .docs-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }
        .docs-card {
            border: 1px solid #e1e8ed;
            border-radius: 6px;
            padding: 20px;
            background: #f8f9fa;
            transition: transform 0.2s;
        }
        .docs-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
        }
        .docs-card h3 {
            margin-top: 0;
            color: #2c3e50;
        }
        .docs-card a {
            display: inline-block;
            background: #3498db;
            color: white;
            padding: 8px 16px;
            text-decoration: none;
            border-radius: 4px;
            margin-top: 10px;
            transition: background-color 0.2s;
        }
        .docs-card a:hover {
            background: #2980b9;
        }
        .spec-links {
            margin-top: 15px;
        }
        .spec-links a {
            background: #95a5a6;
            margin-right: 10px;
            padding: 6px 12px;
            font-size: 14px;
        }
        .spec-links a:hover {
            background: #7f8c8d;
        }
        .badge {
            display: inline-block;
            padding: 3px 6px;
            font-size: 12px;
            border-radius: 3px;
            margin-left: 8px;
        }
        .badge-manual {
            background: #27ae60;
            color: white;
        }
        .badge-generated {
            background: #e67e22;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Go Template API Documentation</h1>
        
        <p>Welcome to the Go Template API documentation. This API is built with Domain-Driven Design principles and provides comprehensive functionality for user management and examples.</p>
        
        <h2>📖 Documentation Interfaces</h2>
        
        <div class="docs-grid">
            <div class="docs-card">
                <h3>📘 Redoc (Manual Spec) <span class="badge badge-manual">OpenAPI 3.x</span></h3>
                <p>Modern, clean documentation interface with the manually crafted OpenAPI 3.x specification. Best for detailed API exploration.</p>
                <a href="/docs/redoc.html">View Redoc Docs</a>
                <div class="spec-links">
                    <a href="/docs/openapi.yaml">📄 YAML Spec</a>
                </div>
            </div>
            
            <div class="docs-card">
                <h3>📙 Swagger UI (Manual Spec) <span class="badge badge-manual">OpenAPI 3.x</span></h3>
                <p>Interactive documentation interface with the manually crafted specification. Try out API calls directly from the browser.</p>
                <a href="/docs/swagger-ui.html">View Swagger UI</a>
                <div class="spec-links">
                    <a href="/docs/openapi.yaml">📄 YAML Spec</a>
                </div>
            </div>
        </div>

        <h2>🔧 Generated Documentation</h2>
        
        <div class="docs-grid">
            <div class="docs-card">
                <h3>⚡ Generated Swagger UI <span class="badge badge-generated">Swagger 2.0</span></h3>
                <p>Automatically generated from Go code annotations. Always reflects the current codebase implementation.</p>
                <a href="/swagger/">View Generated Docs</a>
                <div class="spec-links">
                    <a href="/docs/openapi-generated.yaml">📄 YAML Spec</a>
                    <a href="/docs/openapi-generated.json">📄 JSON Spec</a>
                </div>
            </div>
        </div>
        
        <h2>🛠 API Information</h2>
        <ul>
            <li><strong>Base URL:</strong> <code>http://localhost:8080</code></li>
            <li><strong>Version:</strong> 1.0.0</li>
            <li><strong>Authentication:</strong> Bearer JWT Token</li>
            <li><strong>Content Type:</strong> application/json</li>
        </ul>
        
        <h2>🔗 Quick Links</h2>
        <ul>
            <li><a href="/health">🩺 Health Check</a></li>
            <li><a href="/api/v1/auth/register">🔐 User Registration</a> (POST)</li>
            <li><a href="/api/v1/auth/login">🔑 User Login</a> (POST)</li>
            <li><a href="/admin/v1/login">👑 Admin Login</a> (POST)</li>
        </ul>
        
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #e1e8ed;">
        
        <p style="text-align: center; color: #7f8c8d; font-size: 14px;">
            Built with ❤️ using Go, Chi Router, and Domain-Driven Design principles
        </p>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(indexHTML))
}

// redirectToRedocHTML redirects to the redoc HTML file
func (h *Handler) redirectToRedocHTML(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/redoc.html", http.StatusFound)
}

// redirectToSwaggerUI redirects to the swagger UI HTML file
func (h *Handler) redirectToSwaggerUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/swagger-ui.html", http.StatusFound)
}

// ServeFile serves a specific file from the embedded filesystem
func (h *Handler) ServeFile(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(h.docsFS, filename)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Set appropriate content type
		switch {
		case strings.HasSuffix(filename, ".yaml"), strings.HasSuffix(filename, ".yml"):
			w.Header().Set("Content-Type", "application/x-yaml")
		case strings.HasSuffix(filename, ".json"):
			w.Header().Set("Content-Type", "application/json")
		case strings.HasSuffix(filename, ".html"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
