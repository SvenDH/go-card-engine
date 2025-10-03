package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// LoginUser represents a user logging in
// TODO: Move this to the game package
// TODO: Implement proper user authentication
// TODO: Add input validation
// TODO: Add rate limiting
// TODO: Add CSRF protection
// TODO: Add secure cookie handling
// TODO: Implement proper password hashing
// TODO: Add user session management
// TODO: Add JWT or similar token-based auth
// TODO: Add refresh token support
// TODO: Add password reset functionality
// TODO: Add email verification
// TODO: Add 2FA support
// TODO: Add account lockout after failed attempts
// TODO: Add password complexity requirements
// TODO: Add audit logging for security events
type LoginUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TODO: Add proper error responses with consistent format
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
	if err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

type Cors struct {
	handler http.Handler
}
func (c *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.handler.ServeHTTP(w, r)
}

type Logger struct {
    handler http.Handler
	logger *log.Logger
}
func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    l.handler.ServeHTTP(w, r)
    l.logger.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
}

type Router struct {
	addr     string
	broker   Broker
	repo     *Repository
	wsServer *Server
	mux      http.Handler
}

// NewRouter creates a new HTTP router with all the necessary routes
// TODO: Add middleware for logging, recovery, CORS, etc.
// TODO: Add request timeouts
// TODO: Add request/response logging
// TODO: Add metrics collection
// TODO: Add health check endpoint
// TODO: Add graceful shutdown handling
// TODO: Add request ID to context for tracing
// TODO: Add rate limiting middleware
// TODO: Add compression middleware
// TODO: Add security headers middleware
func NewRouter(addr string, broker Broker, repo *Repository, wsServer *Server) *Router {
	mux := http.NewServeMux()
	// WebSocket endpoint for real-time game updates
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Add authentication middleware
		// TODO: Add rate limiting
		// TODO: Add connection limits per IP
		// TODO: Add request validation
		// TODO: Add CORS support if needed
		// TODO: Add WebSocket subprotocol negotiation
		// TODO: Add ping/pong handling
		// TODO: Add message size limits
		// TODO: Add connection timeouts
		// TODO: Add error handling
		ServeWs(wsServer, w, r)
	})
	
	// Login endpoint for user authentication
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
			return
		}

		var loginUser LoginUser
		if err := json.NewDecoder(r.Body).Decode(&loginUser); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Basic validation
		if loginUser.Username == "" || loginUser.Password == "" {
			respondWithError(w, http.StatusBadRequest, "Username and password are required")
			return
		}

		// TODO: Implement proper user authentication
		// This is a temporary implementation that accepts any credentials
		// In a real application, you would:
		// 1. Look up the user in the database
		// 2. Verify the password hash
		// 3. Generate a session token
		// 4. Set secure HTTP-only cookies

		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"user": map[string]string{
				"username": loginUser.Username,
			},
			// TODO: Add proper token generation
			// "token": "your-jwt-token-here",
		})
	})

	// Register endpoint for new user registration
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
			return
		}

		var user LoginUser
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Basic validation
		if user.Username == "" || user.Password == "" {
			respondWithError(w, http.StatusBadRequest, "Username and password are required")
			return
		}

		// TODO: Implement proper user registration
		// This is a temporary implementation that accepts any registration
		// In a real application, you would:
		// 1. Validate username format and uniqueness
		// 2. Validate password strength
		// 3. Hash the password
		// 4. Create the user in the database
		// 5. Send a verification email

		respondWithJSON(w, http.StatusCreated, map[string]string{
			"status":  "success",
			"message": "User registered successfully",
			// TODO: Add email verification info
		})
	})

	// Serve static files from the public directory
	// TODO: Add cache control headers
	// TODO: Add gzip/brotli compression
	// TODO: Add security headers
	// TODO: Add SPA fallback for client-side routing
	// TODO: Add file system caching
	fs := http.FileServer(http.Dir("./public"))
	mux.Handle("/", fs)

	logger := log.New(os.Stderr, "[http]: ", log.LstdFlags)
	return &Router{
		addr:     addr,
		broker:   broker,
		repo:     repo,
		wsServer: wsServer,
		mux:      &Logger{&Cors{mux}, logger},
	}
}

func (r *Router) Run() {
	go r.wsServer.Run()
	log.Printf("http server started on %s", r.addr)
	log.Fatal(http.ListenAndServe(r.addr, r.mux))
}