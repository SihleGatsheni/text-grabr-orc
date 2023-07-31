package main

import (
	"encoding/json"
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	http.HandleFunc("/api/v1/extract-text", handleCORS(handleOCR))

	http.HandleFunc("/", handleServerStatus)

	port := ":8080"
	log.Println("Server started on port", port)
	http.ListenAndServe(port, nil)
}

func handleServerStatus(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	server := ServerResult{
		Host:   host,
		Status: "Running....",
	}
	responseJSON, _ := json.Marshal(server)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func handleOCR(w http.ResponseWriter, r *http.Request) {
	// Parse the form data to get the uploaded file
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit for the uploaded file
	if err != nil {
		http.Error(w, "Unable to process the form", http.StatusBadRequest)
		return
	}

	// Get the uploaded file from the form data
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the uploaded file to a temporary location
	tempFile := fmt.Sprintf("temp_%s", handler.Filename)
	out, err := os.Create(tempFile)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile)

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Check the file extension (image or PDF)
	ext := strings.ToLower(filepath.Ext(handler.Filename))
	var text string
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
		text, err = extractTextFromImage(tempFile)
	} else {
		http.Error(w, "Invalid file format", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Failed to extract text", http.StatusInternalServerError)
		return
	}

	result := processText(text)
	log.Printf("extracting text from image....")
	responseJSON, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "Failed to create JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func extractTextFromImage(filePath string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()

	// Pre-process the image before passing it to Tesseract
	processedImage, err := preprocessImage(filePath)
	if err != nil {
		return "", err
	}

	// Save the preprocessed image to a temporary location
	tempImageFile := fmt.Sprintf("temp_%s", "processed_"+filepath.Base(filePath))
	err = saveImage(processedImage, tempImageFile)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempImageFile)

	client.SetImage(tempImageFile)
	client.SetLanguage("eng")
	client.SetPageSegMode(gosseract.PSM_AUTO_OSD) // Set page segmentation mode

	return client.Text()
}

func preprocessImage(filePath string) (image.Image, error) {
	srcImage, err := loadImage(filePath)
	if err != nil {
		return nil, err
	}

	// Convert the image to grayscale
	grayImage := image.NewGray(srcImage.Bounds())
	for x := 0; x < grayImage.Rect.Max.X; x++ {
		for y := 0; y < grayImage.Rect.Max.Y; y++ {
			grayImage.Set(x, y, srcImage.At(x, y))
		}
	}

	binarizedImage := binarize(grayImage)
	return binarizedImage, nil
}

func loadImage(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func saveImage(img image.Image, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	err = png.Encode(out, img)
	if err != nil {
		return err
	}

	return nil
}

func binarize(img *image.Gray) *image.NRGBA {
	threshold := uint8(128)
	bounds := img.Bounds()
	binarized := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := img.GrayAt(x, y)
			var newPixel color.NRGBA
			if pixel.Y < threshold {
				newPixel = color.NRGBA{0, 0, 0, 255} // Black
			} else {
				newPixel = color.NRGBA{255, 255, 255, 255} // White
			}
			binarized.SetNRGBA(x, y, newPixel)
		}
	}
	return binarized
}

func processText(text string) Result {
	return Result{
		Text:      text,
		TextCount: len(text),
	}
}

func handleCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			return
		}

		// Call the next handler
		next(w, r)
	}
}

type Result struct {
	Text      string
	TextCount int
}

type ServerResult struct {
	Host   string `json:"host"`
	Status string `json:"status"`
}
