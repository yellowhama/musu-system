package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CheckTesseract verifies if the tesseract binary is available in the system PATH.
func CheckTesseract() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

// RunOCR executes Tesseract OCR on the given image file path and returns the extracted text.
func RunOCR(imagePath string) (string, error) {
	if !CheckTesseract() {
		return "", fmt.Errorf("tesseract binary not found in PATH")
	}

	// Use 'stdout' to get the result directly into a buffer
	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", "eng+kor")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract error: %v (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}
