package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var uploadPath = "uploads"

// Template for the HTML form
var tmpl = template.Must(template.ParseFiles("templates/index.html"))

// Image data struct for tracking uploaded images
type ImageData struct {
	Image1    string
	Image2    string
	Image3    string
	Output    string
	Timestamp int64
}

// Main handler for the form
func formHandler(w http.ResponseWriter, r *http.Request) {
	images := ImageData{
		Image1:    getImagePath("image1"),
		Image2:    getImagePath("image2"),
		Image3:    getImagePath("image3"),
		Timestamp: time.Now().Unix(),
	}
	tmpl.Execute(w, images)
}

// Helper function to get image path if it exists
func getImagePath(name string) string {
	path := filepath.Join(uploadPath, name+".png")
	if _, err := os.Stat(path); err == nil {
		return "/" + path // Return relative path for use in HTML
	}
	return ""
}

// Upload handler that only accepts PNG files
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // Limit upload size to 10MB
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	// Check which image field was submitted and process accordingly
	imageField := r.URL.Query().Get("image")
	if imageField != "image1" && imageField != "image2" && imageField != "image3" {
		http.Error(w, "Invalid image field", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile(imageField)
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if the uploaded file is a PNG
	if !isPNG(header) {
		http.Error(w, "Only PNG files are allowed", http.StatusUnsupportedMediaType)
		return
	}

	// Read the first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		http.Error(w, "Unable to read file", http.StatusInternalServerError)
		return
	}

	// Detect the file's MIME type
	mimeType := http.DetectContentType(buffer)
	fmt.Printf("Uploaded File: %s\n", header.Filename)
	fmt.Printf("MIME Type: %s\n", mimeType)

	// Check if the MIME type is an allowed image type
	if mimeType != "image/png" && mimeType != "image/jpeg" {
		http.Error(w, "Invalid file type. Only PNG and JPEG are allowed.", http.StatusUnsupportedMediaType)
		return
	}

	// (Optional) Reset the file pointer to the beginning
	_, err = file.Seek(0, 0) // Rewind to the beginning of the file
	if err != nil {
		http.Error(w, "Unable to seek file", http.StatusInternalServerError)
		return
	}


	// Save the file with a fixed .png extension
	filename := fmt.Sprintf("%s.png", imageField)
	filepath := filepath.Join(uploadPath, filename)

	out, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Unable to save the file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		http.Error(w, "Unable to save the file", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Helper function to check if the uploaded file is a PNG
func isPNG(fileHeader *multipart.FileHeader) bool {
	return strings.EqualFold(filepath.Ext(fileHeader.Filename), ".png")
}

// Run script handler
func runScriptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Run your script
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	scriptPath := filepath.Join(currentDir, "scripts", "script.sh")
	cmd := exec.Command(scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Script failed: %v\nOutput: %s", err, output), http.StatusInternalServerError)
		return
	}

	// Return the script output as plain text
	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}

func main() {
	// Ensure the upload path exists
	os.MkdirAll(uploadPath, os.ModePerm)

	// Serve static files from uploads directory
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	http.HandleFunc("/", formHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/run-script", runScriptHandler)

	fmt.Println("Server starting at http://localhost:8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
