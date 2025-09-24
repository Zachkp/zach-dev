// main.go - Updated to use separate admin module
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	_ "modernc.org/sqlite"

	"github.com/gin-gonic/gin"
)

// Database connection
var db *sql.DB

func main() {
	// Initialize database and admin systems
	initDB()
	initVisitorTracking() // from admin.go
	initAdminToken()      // from admin.go
	defer db.Close()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// Configure trusted proxies for Render.com
	if gin.Mode() == gin.ReleaseMode {
		// Production: Trust only Render's proxy network
		r.SetTrustedProxies([]string{
			"10.0.0.0/8",     // Private network
			"172.16.0.0/12",  // Docker networks
			"192.168.0.0/16", // Private network
		})
	} else {
		// Development: Trust localhost
		r.SetTrustedProxies([]string{"127.0.0.1"})
	}

	// Add visitor tracking middleware (from admin.go)
	r.Use(visitorTrackingMiddleware())

	// Add https redirect for custom domain
	r.Use(httpsRedirectMiddleware())

	r.Static("/images", "./images")
	r.Static("/static", "./static")

	// Setup admin routes (from admin.go)
	setupAdminRoutes(r)

	// Your existing routes...
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"aboutMeContent":      AboutMe,
			"projectOneContent":   ProjectOne,
			"projectTwoContent":   ProjectTwo,
			"projectThreeContent": ProjectThree,
			"projectFourContent":  ProjectFour,
		})
	})

	// HTMX Contact form endpoint
	r.GET("/contact-form", func(c *gin.Context) {
		c.HTML(http.StatusOK, "contact.html", gin.H{
			"title": "Contact Me",
		})
	})

	// HTMX Url Shortener endpoint
	r.GET("/url-shortener", func(c *gin.Context) {
		c.HTML(http.StatusOK, "urlShort.html", gin.H{
			"title": "URL Shortener",
		})
	})

	// Handle URL shortening form submission
	r.POST("/shorten-url", func(c *gin.Context) {
		originalURL := strings.TrimSpace(c.PostForm("originalUrl"))

		// Validate URL
		if originalURL == "" {
			c.HTML(http.StatusOK, "url-shortener-error.html", gin.H{
				"error": "Please enter a URL to shorten.",
			})
			return
		}

		// Parse and validate URL format
		parsedURL, err := url.Parse(originalURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			c.HTML(http.StatusOK, "url-shortener-error.html", gin.H{
				"error": "Please enter a valid URL starting with http:// or https://",
			})
			return
		}

		// Generate short code
		shortCode, err := generateShortCode()
		if err != nil {
			c.HTML(http.StatusOK, "url-shortener-error.html", gin.H{
				"error": "Sorry, there was an error generating the short URL. Please try again.",
			})
			return
		}

		// Save to database
		err = saveURL(shortCode, originalURL)
		if err != nil {
			log.Printf("Error saving URL: %v", err)
			c.HTML(http.StatusOK, "url-shortener-error.html", gin.H{
				"error": "Sorry, there was an error saving the short URL. Please try again.",
			})
			return
		}

		// Build the shortened URL
		var shortURL string
		if gin.Mode() == gin.DebugMode || strings.Contains(c.Request.Host, "localhost") {
			// Development
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			shortURL = fmt.Sprintf("%s://%s/s/%s", scheme, c.Request.Host, shortCode)
		} else {
			// Production - use your custom domain
			shortURL = fmt.Sprintf("https://zachkp.dev/s/%s", shortCode)
		}

		c.HTML(http.StatusOK, "url-shortener-success.html", gin.H{
			"shortUrl":    shortURL,
			"originalUrl": originalURL,
		})
	})

	// Handle shortened URL redirects (with click tracking)
	r.GET("/s/:code", func(c *gin.Context) {
		shortCode := c.Param("code")

		// Get original URL and increment click count
		originalURL, exists := getURL(shortCode)
		if !exists {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"message": "Short URL not found",
			})
			return
		}

		c.Redirect(http.StatusFound, originalURL)
	})

	// Resume download
	r.GET("/resume", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", "attachment; filename=Zachariah_Kordas_Potter_Resume.pdf")
		c.Header("Content-Type", "application/pdf")
		c.File("./static/Zach Kordas-Potter Resume.pdf")
	})

	// Work experience content
	r.GET("/work-content", func(c *gin.Context) {
		c.HTML(http.StatusOK, "work-content.html", gin.H{
			"jobTitle":  jobTitle,
			"company":   company,
			"startDate": startDateWork,
			"endDate":   endDate,
			"logoPath":  "images/TargetLogo.jpg",
			"bulletPoints": []string{
				targetBullet1,
				targetBullet2,
				targetBullet3,
			},
			"jobTitle2":  jobTitle2,
			"company2":   company2,
			"startDate2": startDateWork2,
			"endDate2":   endDate2,
			"logoPath2":  "images/jasonsCateringLogo.png",
			"bulletPoints2": []string{
				cateringBullet1,
				cateringBullet2,
				cateringBullet3,
			},
		})
	})

	// Education content
	r.GET("/education-content", func(c *gin.Context) {
		c.HTML(http.StatusOK, "education-content.html", gin.H{
			"degree":      degree,
			"institution": institution,
			"startDate":   startDateEdu,
			"endDate":     endDateEdu,
			"logoPath":    "images/WGU-logo.png",
			"bulletPoints": []string{
				eduBullet1,
				eduBullet2,
				eduBullet3,
			},
			"degree2":      certification,
			"institution2": institution2,
			"startDate2":   startDateEdu2,
			"endDate2":     endDateEdu2,
			"logoPath2":    "images/comptiaCert.png",
			"bulletPoints2": []string{
				certBullet1,
				certBullet2,
				certBullet3,
			},
		})
	})

	// Handle contact form submission
	r.POST("/contact", func(c *gin.Context) {
		name := c.PostForm("fullName")
		email := c.PostForm("email")
		message := c.PostForm("message")

		err := sendContactEmail(name, email, message)
		if err != nil {
			c.HTML(http.StatusOK, "contact-error.html", gin.H{
				"error": "Sorry, there was an error sending your message. Please try again later.",
			})
			return
		}

		c.HTML(http.StatusOK, "contact-success.html", gin.H{
			"success": "Thank you for your message! I'll get back to you soon.",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func httpsRedirectMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// In production, force HTTPS
		if gin.Mode() == gin.ReleaseMode {
			if c.GetHeader("X-Forwarded-Proto") == "http" {
				httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
				c.Redirect(http.StatusMovedPermanently, httpsURL)
				c.Abort()
				return
			}
		}
		c.Next()
	})
}

// Database initialization
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./urls.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS urls (
		short_code TEXT PRIMARY KEY,
		original_url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	log.Println("Database initialized successfully")
}

// Save URL to database
func saveURL(shortCode, originalURL string) error {
	_, err := db.Exec("INSERT INTO urls (short_code, original_url) VALUES (?, ?)", shortCode, originalURL)
	return err
}

// Get URL and track clicks (enhanced for admin)
func getURL(shortCode string) (string, bool) {
	var originalURL string
	err := db.QueryRow("SELECT original_url FROM urls WHERE short_code = ?", shortCode).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		log.Printf("Database error: %v", err)
		return "", false
	}

	// Increment click count in background
	go func() {
		_, err := db.Exec("UPDATE urls SET clicks = COALESCE(clicks, 0) + 1 WHERE short_code = ?", shortCode)
		if err != nil {
			log.Printf("Error updating click count: %v", err)
		}
	}()

	return originalURL, true
}

// Generate random short code
func generateShortCode() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	shortCode := base64.URLEncoding.EncodeToString(bytes)
	shortCode = strings.TrimRight(shortCode, "=")
	if len(shortCode) > 8 {
		shortCode = shortCode[:8]
	}

	return shortCode, nil
}

// Send contact email
func sendContactEmail(name, email, message string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	toEmail := os.Getenv("TO_EMAIL")

	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if toEmail == "" {
		toEmail = "zachkordaspotter@gmail.com"
	}

	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	subject := fmt.Sprintf("Portfolio Contact: %s", name)
	body := fmt.Sprintf(`
New contact form submission from your portfolio:

Name: %s
Email: %s
Message:
%s

---
Sent from your portfolio contact form
`, name, email, message)

	msg := []byte("To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"From: " + smtpUser + "\r\n" +
		"Reply-To: " + email + "\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUser, []string{toEmail}, msg)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return err
	}

	fmt.Printf("Email sent successfully from %s (%s)\n", name, email)
	return nil
}
