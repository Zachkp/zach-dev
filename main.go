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

	// Work experience content
	// In your /work-content route
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
		})
	})

	// Education content
	// In your /education-content route
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
