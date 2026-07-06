// flashcards.go - CSV-to-flashcards tool for the site
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Flashcard is a single term/definition pair.
type Flashcard struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

func setupFlashcardRoutes(r *gin.Engine) {
	// Full standalone flashcards page
	r.GET("/flashcards", func(c *gin.Context) {
		c.HTML(http.StatusOK, "flashcards.html", gin.H{})
	})

	// Upload form partial, loaded into the page via HTMX (initial load, reset, retry)
	r.GET("/flashcards-form", func(c *gin.Context) {
		c.HTML(http.StatusOK, "flashcards-form.html", gin.H{})
	})

	// Load a built-in sample deck so people can try it without a file
	r.GET("/flashcards-sample", func(c *gin.Context) {
		sample := []Flashcard{
			{Front: "Mitochondria", Back: "The organelle that produces most of a cell's ATP energy"},
			{Front: "Photosynthesis", Back: "Process plants use to convert sunlight into chemical energy"},
			{Front: "Osmosis", Back: "Movement of water across a membrane from low to high solute concentration"},
			{Front: "Ecosystem", Back: "A community of organisms interacting with their physical environment"},
			{Front: "Homeostasis", Back: "The maintenance of stable internal conditions in an organism"},
		}
		renderFlashcardViewer(c, "Sample deck", sample)
	})

	// Handle the uploaded CSV
	r.POST("/parse-flashcards-csv", handleFlashcardUpload)
}

func handleFlashcardUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("csvFile")
	if err != nil {
		c.HTML(http.StatusOK, "flashcards-error.html", gin.H{
			"error": "Please choose a CSV file to upload.",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.HTML(http.StatusOK, "flashcards-error.html", gin.H{
			"error": "Sorry, that file couldn't be opened. Please try again.",
		})
		return
	}
	defer file.Close()

	cards, parseErr := parseFlashcardCSV(file)
	if parseErr != nil {
		c.HTML(http.StatusOK, "flashcards-error.html", gin.H{
			"error": parseErr.Error(),
		})
		return
	}

	renderFlashcardViewer(c, fileHeader.Filename, cards)
}

// parseFlashcardCSV reads rows of at least two columns, optionally skipping
// a header row like "term,definition", and returns them as Flashcards.
func parseFlashcardCSV(r io.Reader) ([]Flashcard, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // tolerate ragged rows; we validate manually below
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("sorry, that file couldn't be read as a CSV")
	}

	var filtered [][]string
	for _, row := range rows {
		if len(row) >= 2 && strings.TrimSpace(row[0]) != "" {
			filtered = append(filtered, row)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("couldn't find any rows with two columns in that file")
	}

	// Detect a header row, e.g. "term,definition" or "question,answer"
	frontHeaders := map[string]bool{"term": true, "question": true, "front": true, "word": true}
	backHeaders := map[string]bool{"definition": true, "answer": true, "back": true, "meaning": true}

	first := filtered[0]
	dataRows := filtered
	if frontHeaders[strings.ToLower(strings.TrimSpace(first[0]))] &&
		backHeaders[strings.ToLower(strings.TrimSpace(first[1]))] {
		dataRows = filtered[1:]
	}

	if len(dataRows) == 0 {
		return nil, fmt.Errorf("that file only has a header row — add some flashcards below it")
	}

	cards := make([]Flashcard, 0, len(dataRows))
	for _, row := range dataRows {
		cards = append(cards, Flashcard{
			Front: strings.TrimSpace(row[0]),
			Back:  strings.TrimSpace(row[1]),
		})
	}

	return cards, nil
}

func renderFlashcardViewer(c *gin.Context, deckName string, cards []Flashcard) {
	cardsJSON, err := json.Marshal(cards)
	if err != nil {
		c.HTML(http.StatusOK, "flashcards-error.html", gin.H{
			"error": "Something went wrong building the deck. Please try again.",
		})
		return
	}

	c.HTML(http.StatusOK, "flashcards-viewer.html", gin.H{
		"deckName":  deckName,
		"cardCount": len(cards),
		"cardsJSON": string(cardsJSON),
	})
}
