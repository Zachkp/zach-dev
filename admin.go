// admin.go
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Admin-related structs
type VisitorMetric struct {
	ID        int       `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	Country   string    `json:"country,omitempty"`
}

type URLStat struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	Clicks      int       `json:"clicks"`
}

type AdminStats struct {
	TotalVisitors    int64           `json:"total_visitors"`
	UniqueVisitors   int64           `json:"unique_visitors"`
	TotalURLs        int64           `json:"total_urls"`
	TotalClicks      int64           `json:"total_clicks"`
	TopURLs          []URLStat       `json:"top_urls"`
	RecentVisitors   []VisitorMetric `json:"recent_visitors"`
	VisitorsToday    int64           `json:"visitors_today"`
	VisitorsThisWeek int64           `json:"visitors_this_week"`
}

var adminToken string

// Initialize admin token
func initAdminToken() {
	adminToken = generateAdminToken()
	log.Printf("Admin access available at: /admin/login")
	// Only show token in development mode
	if gin.Mode() == gin.DebugMode {
		log.Printf("Admin token (dev only): %s", adminToken)
	}
}

func generateAdminToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate admin token:", err)
	}
	return hex.EncodeToString(bytes)
}

// Middleware to check admin authentication
func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("admin_token")
		if err != nil || subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) != 1 {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Middleware to track visitors (non-blocking)
func visitorTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip tracking for static files and admin pages
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/images/") ||
			strings.HasPrefix(path, "/admin/") ||
			strings.HasPrefix(path, "/favicon") {
			c.Next()
			return
		}

		// Track visitor in background goroutine
		go trackVisitor(c.ClientIP(), c.GetHeader("User-Agent"), path)
		c.Next()
	}
}

// Track visitor in background
func trackVisitor(ip, userAgent, path string) {
	_, err := db.Exec(`
		INSERT INTO visitors (ip, user_agent, path, timestamp) 
		VALUES (?, ?, ?, ?)
	`, ip, userAgent, path, time.Now())
	if err != nil {
		log.Printf("Error recording visitor: %v", err)
	}
}

// Initialize visitor tracking table
func initVisitorTracking() {
	createVisitorTable := `
	CREATE TABLE IF NOT EXISTS visitors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ip TEXT NOT NULL,
		user_agent TEXT,
		path TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		country TEXT
	)`

	_, err := db.Exec(createVisitorTable)
	if err != nil {
		log.Fatal("Failed to create visitors table:", err)
	}

	// Add clicks tracking to URLs table if it doesn't exist
	addClicksColumn := `
	ALTER TABLE urls ADD COLUMN clicks INTEGER DEFAULT 0
	`
	// Ignore error if column already exists
	db.Exec(addClicksColumn)

	log.Println("Visitor tracking initialized")
}

// Get comprehensive admin statistics
func getAdminStats() (*AdminStats, error) {
	stats := &AdminStats{}

	// Total visitors
	err := db.QueryRow("SELECT COUNT(*) FROM visitors").Scan(&stats.TotalVisitors)
	if err != nil {
		return nil, err
	}

	// Unique visitors (by IP)
	err = db.QueryRow("SELECT COUNT(DISTINCT ip) FROM visitors").Scan(&stats.UniqueVisitors)
	if err != nil {
		return nil, err
	}

	// Total URLs
	err = db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&stats.TotalURLs)
	if err != nil {
		return nil, err
	}

	// Total clicks
	err = db.QueryRow("SELECT COALESCE(SUM(clicks), 0) FROM urls").Scan(&stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	// Visitors today
	err = db.QueryRow(`
		SELECT COUNT(*) FROM visitors 
		WHERE DATE(timestamp) = DATE('now')
	`).Scan(&stats.VisitorsToday)
	if err != nil {
		return nil, err
	}

	// Visitors this week
	err = db.QueryRow(`
		SELECT COUNT(*) FROM visitors 
		WHERE timestamp >= datetime('now', '-7 days')
	`).Scan(&stats.VisitorsThisWeek)
	if err != nil {
		return nil, err
	}

	// Top URLs by clicks
	rows, err := db.Query(`
		SELECT short_code, original_url, created_at, COALESCE(clicks, 0) as clicks
		FROM urls 
		ORDER BY clicks DESC, created_at DESC 
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var url URLStat
		err := rows.Scan(&url.ShortCode, &url.OriginalURL, &url.CreatedAt, &url.Clicks)
		if err != nil {
			continue
		}
		stats.TopURLs = append(stats.TopURLs, url)
	}

	// Recent visitors
	rows, err = db.Query(`
		SELECT id, ip, user_agent, path, timestamp
		FROM visitors 
		ORDER BY timestamp DESC 
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var visitor VisitorMetric
		err := rows.Scan(&visitor.ID, &visitor.IP, &visitor.UserAgent, &visitor.Path, &visitor.Timestamp)
		if err != nil {
			continue
		}
		stats.RecentVisitors = append(stats.RecentVisitors, visitor)
	}

	return stats, nil
}

// Setup all admin routes
func setupAdminRoutes(r *gin.Engine) {
	// Admin login page
	r.GET("/admin/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin-login.html", gin.H{
			"title": "Admin Login",
		})
	})

	// Admin login handler
	r.POST("/admin/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		// Get credentials from environment variables
		adminUsername := os.Getenv("ADMIN_USERNAME")
		adminPassword := os.Getenv("ADMIN_PASSWORD")

		// Default credentials for development (remove in production)
		if adminUsername == "" {
			adminUsername = "admin"
			log.Println("WARNING: Using default admin username. Set ADMIN_USERNAME environment variable.")
		}
		if adminPassword == "" {
			adminPassword = "admin123"
			log.Println("WARNING: Using default admin password. Set ADMIN_PASSWORD environment variable.")
		}

		if username == adminUsername && password == adminPassword {
			// Set secure cookie (24 hours)
			c.SetCookie("admin_token", adminToken, 3600*24, "/admin", "", false, true)
			c.Redirect(http.StatusFound, "/admin/dashboard")
		} else {
			log.Printf("Failed admin login attempt from %s", c.ClientIP())
			c.HTML(http.StatusUnauthorized, "admin-login.html", gin.H{
				"error": "Invalid credentials",
			})
		}
	})

	// Admin logout
	r.GET("/admin/logout", func(c *gin.Context) {
		c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
		c.Redirect(http.StatusFound, "/admin/login")
	})

	// Protected admin routes group
	adminGroup := r.Group("/admin")
	adminGroup.Use(adminAuthMiddleware())

	// Admin dashboard
	adminGroup.GET("/dashboard", func(c *gin.Context) {
		stats, err := getAdminStats()
		if err != nil {
			log.Printf("Error loading admin stats: %v", err)
			c.HTML(http.StatusInternalServerError, "admin-error.html", gin.H{
				"error": "Failed to load statistics",
			})
			return
		}

		c.HTML(http.StatusOK, "admin-dashboard.html", gin.H{
			"stats": stats,
		})
	})

	// Admin API endpoints for HTMX/AJAX
	adminGroup.GET("/api/stats", func(c *gin.Context) {
		stats, err := getAdminStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stats)
	})

	// View all URLs
	adminGroup.GET("/urls", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT short_code, original_url, created_at, COALESCE(clicks, 0) as clicks
			FROM urls 
			ORDER BY created_at DESC
		`)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "admin-error.html", gin.H{
				"error": "Failed to load URLs",
			})
			return
		}
		defer rows.Close()

		var urls []URLStat
		for rows.Next() {
			var url URLStat
			err := rows.Scan(&url.ShortCode, &url.OriginalURL, &url.CreatedAt, &url.Clicks)
			if err != nil {
				continue
			}
			urls = append(urls, url)
		}

		c.HTML(http.StatusOK, "admin-urls.html", gin.H{
			"urls": urls,
		})
	})

	// View visitors
	adminGroup.GET("/visitors", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, ip, user_agent, path, timestamp
			FROM visitors 
			ORDER BY timestamp DESC 
			LIMIT 200
		`)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "admin-error.html", gin.H{
				"error": "Failed to load visitors",
			})
			return
		}
		defer rows.Close()

		var visitors []VisitorMetric
		for rows.Next() {
			var visitor VisitorMetric
			err := rows.Scan(&visitor.ID, &visitor.IP, &visitor.UserAgent, &visitor.Path, &visitor.Timestamp)
			if err != nil {
				continue
			}
			visitors = append(visitors, visitor)
		}

		c.HTML(http.StatusOK, "admin-visitors.html", gin.H{
			"visitors": visitors,
		})
	})

	// Delete URL (with confirmation)
	adminGroup.DELETE("/urls/:code", func(c *gin.Context) {
		shortCode := c.Param("code")

		result, err := db.Exec("DELETE FROM urls WHERE short_code = ?", shortCode)
		if err != nil {
			log.Printf("Error deleting URL %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete URL"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}

		log.Printf("URL %s deleted by admin", shortCode)
		c.JSON(http.StatusOK, gin.H{"message": "URL deleted successfully"})
	})
}
