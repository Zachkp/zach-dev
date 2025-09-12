package main

import (
	"fmt"
	"net/http"
	"net/smtp"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.Static("/images", "./images")
	r.Static("/static", "./static")

	// Home page route
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"aboutMeContent":      AboutMe,
			"projectOneContent":   ProjectOne,
			"projectTwoContent":   ProjectTwo,
			"projectThreeContent": ProjectThree,
			"projectFourContent":  ProjectFour,
		})
	})

	// HTMX Contact form endpoint - returns just the form HTML
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

	// Handle contact form submission with HTMX
	r.POST("/contact", func(c *gin.Context) {
		name := c.PostForm("fullName")
		email := c.PostForm("email")
		message := c.PostForm("message")

		// Send email
		err := sendContactEmail(name, email, message)
		if err != nil {
			// Return error message HTML fragment
			c.HTML(http.StatusOK, "contact-error.html", gin.H{
				"error": "Sorry, there was an error sending your message. Please try again later.",
			})
			return
		}

		// Return success message HTML fragment
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

func sendContactEmail(name, email, message string) error {
	// Email configuration - use environment variables for security
	smtpHost := os.Getenv("SMTP_HOST") // e.g., "smtp.gmail.com"
	smtpPort := os.Getenv("SMTP_PORT") // e.g., "587"
	smtpUser := os.Getenv("SMTP_USER") // your email
	smtpPass := os.Getenv("SMTP_PASS") // your app password
	toEmail := os.Getenv("TO_EMAIL")   // where you want to receive emails

	// Default values for development (remove in production)
	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if toEmail == "" {
		toEmail = "zachkordaspotter@gmail.com" // your email
	}

	// Validate required fields
	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	// Create message
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

	// Compose email
	msg := []byte("To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"From: " + smtpUser + "\r\n" +
		"Reply-To: " + email + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// SMTP authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Send email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUser, []string{toEmail}, msg)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return err
	}

	fmt.Printf("Email sent successfully from %s (%s)\n", name, email)
	return nil
}
