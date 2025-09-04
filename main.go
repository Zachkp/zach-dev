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

	// Work experience content
	r.GET("/work-content", func(c *gin.Context) {
		c.HTML(http.StatusOK, "work-content.html", gin.H{
			"jobTitle":  "Presentation Expert",
			"company":   "Target",
			"startDate": "Aug 2023",
			"endDate":   "Present",
			"logoPath":  "images/TargetLogo.jpg",
			"bulletPoints": []string{
				"Executed over 300 merchandising transitions on tight timelines by organizing team workflows and adapting quickly to changing priorities",
				"Boosted operational efficiency by managing backroom inventory processes and streamlining communication between floor and logistics teams",
				"Enhanced pricing and signage accuracy across departments by standardizing daily checks and collaborating cross-functionally",
			},
			"jobTitle2":  "Manager",
			"company2":   "Jasons Catered Events",
			"startDate2": "Aug 2016",
			"endDate2":   "Present",
			"logoPath2":  "images/jasonsCateringLogo.png",
			"bulletPoints2": []string{
				"Improved client satisfaction by coordinating customized menus and ensuring all dietary requirements were accurately met",
				"Supported event technology by troubleshooting AV equipment and managing digital order tracking systems, reducing technical delays and improving communication",
				"Maintained supply inventory and coordinated timely delivery between venues, optimizing resource allocation and minimizing downtime.",
			},
		})
	})

	// Education content
	r.GET("/education-content", func(c *gin.Context) {
		c.HTML(http.StatusOK, "education-content.html", gin.H{
			"degree":      "Bachelor of Computer Science",
			"institution": "Western Governors University",
			"startDate":   "Sept 2019",
			"endDate":     "May 2023",
			"logoPath":    "images/WGU-logo.png",
			"bulletPoints": []string{
				"Graduated Magna Cum Laude with 3.8 GPA",
				"Relevant coursework: Data Structures, Algorithms, Web Development",
				"Senior project: Machine Learning recommendation system",
			},
			"degree2":      "Project Management",
			"institution2": "Comptia",
			"startDate2":   "July 2022",
			"endDate2":     "Present",
			"logoPath2":    "images/comptiaCert.png",
			"bulletPoints2": []string{
				"Certified in agile project management methodology",
				"Verification code: SRRRPGBSWBRQCCDJ",
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
