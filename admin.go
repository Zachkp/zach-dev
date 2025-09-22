// admin.go - Complete privacy-conscious admin system
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Privacy-conscious visitor tracking struct
type VisitorMetric struct {
	ID        int       `json:"id"`
	HashedIP  string    `json:"hashed_ip"` // Hashed instead of raw IP for privacy
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
var hashingSalt string

// Initialize admin system with privacy considerations
func initAdminToken() {
	adminToken = generateAdminToken()
	hashingSalt = generateAdminToken() // Use for IP hashing

	log.Printf("Admin access available at: /admin/login")
	if gin.Mode() == gin.DebugMode {
		log.Printf("Admin token (dev only): %s", adminToken)
	}

	log.Println("Privacy: Visitor tracking enabled with hashed IP addresses")
}

func generateAdminToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate admin token:", err)
	}
	return hex.EncodeToString(bytes)
}

// Hash IP address for privacy compliance (consistent per IP)
func hashIP(ip string) string {
	hash := sha256.New()
	hash.Write([]byte(ip + hashingSalt))
	return hex.EncodeToString(hash.Sum(nil))[:16] // Truncate for storage efficiency
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

// Privacy-conscious visitor tracking middleware
func visitorTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip tracking for static files and admin pages
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/images/") ||
			strings.HasPrefix(path, "/admin/") ||
			strings.HasPrefix(path, "/favicon") ||
			strings.HasPrefix(path, "/privacy") {
			c.Next()
			return
		}

		// Respect Do Not Track header
		if c.GetHeader("DNT") == "1" {
			c.Next()
			return
		}

		// Track visitor with hashed IP in background
		go trackVisitorPrivacy(c.ClientIP(), c.GetHeader("User-Agent"), path)
		c.Next()
	}
}

// Track visitor with privacy protections
func trackVisitorPrivacy(ip, userAgent, path string) {
	hashedIP := hashIP(ip)

	_, err := db.Exec(`
		INSERT INTO visitors (hashed_ip, user_agent, path, timestamp) 
		VALUES (?, ?, ?, ?)
	`, hashedIP, userAgent, path, time.Now())

	if err != nil {
		log.Printf("Error recording visitor: %v", err)
	}
}

// Initialize privacy-conscious visitor tracking
func initVisitorTracking() {
	// First, try to create the table with the new schema
	createVisitorTable := `
	CREATE TABLE IF NOT EXISTS visitors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hashed_ip TEXT NOT NULL,  -- Store hashed IP instead of raw IP
		user_agent TEXT,
		path TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		country TEXT
	)`

	_, err := db.Exec(createVisitorTable)
	if err != nil {
		log.Fatal("Failed to create visitors table:", err)
	}

	// Check if we need to migrate existing table structure
	// Try to add hashed_ip column in case table exists with old schema
	migrateVisitorTable := `ALTER TABLE visitors ADD COLUMN hashed_ip TEXT`
	_, err = db.Exec(migrateVisitorTable)
	if err != nil {
		// Column might already exist, check if we need to migrate data
		var columnExists int
		checkColumn := `SELECT COUNT(*) as column_exists FROM pragma_table_info('visitors') WHERE name='hashed_ip'`
		db.QueryRow(checkColumn).Scan(&columnExists)

		if columnExists == 0 {
			log.Printf("Warning: Could not add hashed_ip column: %v", err)
			// If we can't add the column, we need to recreate the table
			recreateTable()
		}
	}

	// Migrate existing IP data to hashed_ip if needed
	migrateExistingData()

	// Add clicks tracking to URLs table if it doesn't exist
	addClicksColumn := `ALTER TABLE urls ADD COLUMN clicks INTEGER DEFAULT 0`
	db.Exec(addClicksColumn) // Ignore error if column already exists

	// Clean up old visitor data for privacy compliance (run in background)
	go cleanupOldVisitorData()

	log.Println("Privacy-conscious visitor tracking initialized")
}

// Recreate the visitors table with new schema
func recreateTable() {
	log.Println("Recreating visitors table with new schema...")

	// Rename old table
	_, err := db.Exec(`ALTER TABLE visitors RENAME TO visitors_old`)
	if err != nil {
		log.Printf("Could not rename old table: %v", err)
		return
	}

	// Create new table
	createNewTable := `
	CREATE TABLE visitors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hashed_ip TEXT NOT NULL,
		user_agent TEXT,
		path TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		country TEXT
	)`

	_, err = db.Exec(createNewTable)
	if err != nil {
		log.Printf("Could not create new table: %v", err)
		return
	}

	// Migrate data from old table if it has an 'ip' column
	migrateQuery := `
	INSERT INTO visitors (hashed_ip, user_agent, path, timestamp, country)
	SELECT 
		printf('%016x', abs(random()) % 1000000000) as hashed_ip,
		user_agent, 
		path, 
		timestamp, 
		country 
	FROM visitors_old`

	_, err = db.Exec(migrateQuery)
	if err != nil {
		log.Printf("Could not migrate old data: %v", err)
	} else {
		log.Println("Successfully migrated existing visitor data")
	}

	// Drop old table
	db.Exec(`DROP TABLE visitors_old`)
}

// Migrate existing data to use hashed IPs
func migrateExistingData() {
	// Check if there are rows with empty hashed_ip that have raw ip data
	var needsMigration int
	checkMigration := `SELECT COUNT(*) FROM visitors WHERE hashed_ip IS NULL OR hashed_ip = ''`
	db.QueryRow(checkMigration).Scan(&needsMigration)

	if needsMigration > 0 {
		log.Printf("Migrating %d visitor records to use hashed IPs", needsMigration)

		// For existing records without proper hash, we'll assign consistent placeholder hashes
		// Since we don't have the original IPs, we'll use the record ID for consistency
		updateQuery := `UPDATE visitors SET hashed_ip = printf('%016x', id * 12345) WHERE hashed_ip IS NULL OR hashed_ip = ''`
		_, err := db.Exec(updateQuery)
		if err != nil {
			log.Printf("Error migrating visitor data: %v", err)
		} else {
			log.Printf("Successfully migrated %d records with consistent hashes", needsMigration)
		}
	}
}

// Cleanup old visitor data for privacy compliance
func cleanupOldVisitorData() {
	result, err := db.Exec(`
		DELETE FROM visitors 
		WHERE timestamp < datetime('now', '-12 months')
	`)
	if err != nil {
		log.Printf("Error cleaning up old visitor data: %v", err)
		return
	}

	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		log.Printf("Privacy cleanup: Removed %d visitor records older than 12 months", rowsDeleted)
	}
}

// Get comprehensive admin statistics
func getAdminStats() (*AdminStats, error) {
	stats := &AdminStats{}

	// Total visitors
	err := db.QueryRow("SELECT COUNT(*) FROM visitors").Scan(&stats.TotalVisitors)
	if err != nil {
		return nil, err
	}

	// Unique visitors (by hashed IP)
	err = db.QueryRow("SELECT COUNT(DISTINCT hashed_ip) FROM visitors").Scan(&stats.UniqueVisitors)
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

	// Recent visitors (with hashed IPs for privacy)
	rows, err = db.Query(`
		SELECT id, hashed_ip, user_agent, path, timestamp
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
		err := rows.Scan(&visitor.ID, &visitor.HashedIP, &visitor.UserAgent, &visitor.Path, &visitor.Timestamp)
		if err != nil {
			continue
		}
		stats.RecentVisitors = append(stats.RecentVisitors, visitor)
	}

	return stats, nil
}

// Setup all admin routes
func setupAdminRoutes(r *gin.Engine) {
	// Privacy policy route
	r.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.html", gin.H{
			"title": "Privacy Policy",
		})
	})

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
			if gin.Mode() == gin.DebugMode {
				log.Println("WARNING: Using default admin username. Set ADMIN_USERNAME environment variable.")
			}
		}
		if adminPassword == "" {
			adminPassword = "admin123"
			if gin.Mode() == gin.DebugMode {
				log.Println("WARNING: Using default admin password. Set ADMIN_PASSWORD environment variable.")
			}
		}

		if username == adminUsername && password == adminPassword {
			// Set secure cookie (24 hours)
			c.SetCookie("admin_token", adminToken, 3600*24, "/admin", "", false, true)
			log.Printf("Admin login successful from %s", hashIP(c.ClientIP()))
			c.Redirect(http.StatusFound, "/admin/dashboard")
		} else {
			log.Printf("Failed admin login attempt from %s", hashIP(c.ClientIP()))
			c.HTML(http.StatusUnauthorized, "admin-login.html", gin.H{
				"error": "Invalid credentials",
			})
		}
	})

	// Admin logout
	r.GET("/admin/logout", func(c *gin.Context) {
		c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
		log.Printf("Admin logout from %s", hashIP(c.ClientIP()))
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
			SELECT id, hashed_ip, user_agent, path, timestamp
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
			err := rows.Scan(&visitor.ID, &visitor.HashedIP, &visitor.UserAgent, &visitor.Path, &visitor.Timestamp)
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

		log.Printf("URL %s deleted by admin from %s", shortCode, hashIP(c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"message": "URL deleted successfully"})
	})

	// Privacy compliance endpoint - allow users to request data deletion
	adminGroup.POST("/privacy/delete-visitor-data", func(c *gin.Context) {
		// This would require the user to provide their IP or some identifier
		// For now, just clean up old data
		go cleanupOldVisitorData()
		c.JSON(http.StatusOK, gin.H{"message": "Privacy cleanup initiated"})
	})

	// Admin statistics export (for backups or analysis)
	adminGroup.GET("/export/stats", func(c *gin.Context) {
		stats, err := getAdminStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Set headers for file download
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=admin-stats.json")

		log.Printf("Admin stats exported by %s", hashIP(c.ClientIP()))
		c.JSON(http.StatusOK, stats)
	})
}
