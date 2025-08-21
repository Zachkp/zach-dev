package main

import (
	"net/http"
	"os"

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

	// Handle contact form submission with HTMX
	r.POST("/contact", func(c *gin.Context) {
		name := c.PostForm("fullName")
		email := c.PostForm("email")
		message := c.PostForm("message")

		//TODO: Add actual email func
		println("Contact form submitted:")
		println("Name:", name)
		println("Email:", email)
		println("Message:", message)

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
