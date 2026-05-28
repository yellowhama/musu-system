package harvester

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/ledongthuc/pdf"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type ArxivFetcher struct{}

type ArxivFeed struct {
	Entry ArxivEntry `xml:"entry"`
}

type ArxivEntry struct {
	Title   string   `xml:"title"`
	Summary string   `xml:"summary"`
	Author  []string `xml:"author>name"`
	Link    []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
}

func (f *ArxivFetcher) Fetch(arxivID string) (string, string, error) {
	// 1. Fetch Metadata via Arxiv API
	apiURL := fmt.Sprintf("http://export.arxiv.org/api/query?id_list=%s", arxivID)
	body, _, err := utils.GetWithRetry(apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch metadata: %v", err)
	}

	var feed ArxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return "", "", fmt.Errorf("failed to decode XML: %v", err)
	}

	entry := feed.Entry
	if entry.Title == "" {
		return "", "", fmt.Errorf("paper not found or empty title")
	}

	// 2. High-Quality Content Fetch: Try HTML (ar5iv or official)
	contentMD := ""
	contentSource := ""

	// Attempt ar5iv
	ar5ivURL := fmt.Sprintf("https://ar5iv.org/html/%s", arxivID)
	htmlBody, status, err := utils.GetWithRetry(ar5ivURL, nil)
	if err == nil && status == 200 {
		markdown, mdErr := htmltomarkdown.ConvertString(string(htmlBody))
		if mdErr == nil {
			contentMD = markdown
			contentSource = "ar5iv HTML"
		}
	}

	// Attempt official Arxiv HTML if ar5iv failed
	if contentMD == "" {
		officialURL := fmt.Sprintf("https://arxiv.org/html/%s", arxivID)
		htmlBody, status, err = utils.GetWithRetry(officialURL, nil)
		if err == nil && status == 200 {
			markdown, mdErr := htmltomarkdown.ConvertString(string(htmlBody))
			if mdErr == nil {
				contentMD = markdown
				contentSource = "Arxiv HTML"
			}
		}
	}

	// 3. Fallback: PDF text extraction
	if contentMD == "" {
		pdfURL := ""
		for _, link := range entry.Link {
			if link.Type == "application/pdf" {
				pdfURL = link.Href
				break
			}
		}

		if pdfURL != "" {
			pdfURL = strings.Replace(pdfURL, "http://", "https://", 1)
			pdfText, pdfErr := f.extractPDFText(pdfURL)
			if pdfErr == nil && len(pdfText) > 200 {
				contentMD = pdfText
				contentSource = "PDF (standard extraction)"
			} else {
				// 4. Ultimate Fallback: OCR
				if utils.CheckTesseract() {
					fmt.Printf("   🧩 Standard extraction failed or result too short. Triggering OCR fallback for %s...\n", arxivID)
					ocrText, ocrErr := f.performPDFOCR(pdfURL)
					if ocrErr == nil && ocrText != "" {
						contentMD = ocrText
						contentSource = "PDF (OCR)"
					}
				}
			}
		}
	}

	// 5. Build Markdown
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", strings.TrimSpace(entry.Title)))
	sb.WriteString(fmt.Sprintf("**Authors:** %s\n\n", strings.Join(entry.Author, ", ")))
	sb.WriteString("## Abstract\n\n")
	sb.WriteString(strings.TrimSpace(entry.Summary))
	sb.WriteString("\n\n---\n\n")

	if contentMD != "" {
		sb.WriteString(fmt.Sprintf("## Paper Content (Source: %s)\n\n", contentSource))
		sb.WriteString(contentMD)
	} else {
		sb.WriteString("> *Note: Full paper content extraction was not available or failed.*\n")
	}

	return entry.Title, sb.String(), nil
}

func (f *ArxivFetcher) performPDFOCR(pdfURL string) (string, error) {
	// Download PDF to temp file
	body, _, err := utils.GetWithRetry(pdfURL, nil)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "musu-ocr-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	pdfPath := filepath.Join(tempDir, "target.pdf")
	if err := os.WriteFile(pdfPath, body, 0644); err != nil {
		return "", err
	}

	// Extract images using pdfcpu
	imgDir := filepath.Join(tempDir, "images")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image dir: %v", err)
	}
	if err := pdfapi.ExtractImagesFile(pdfPath, imgDir, nil, nil); err != nil {
		return "", fmt.Errorf("pdfcpu image extraction failed: %v", err)
	}

	// Run OCR on all extracted images
	var fullText strings.Builder
	files, _ := os.ReadDir(imgDir)
	if len(files) == 0 {
		return "", fmt.Errorf("no images extracted from PDF")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fmt.Printf("      🔍 OCRing page image: %s\n", file.Name())
		text, err := utils.RunOCR(filepath.Join(imgDir, file.Name()))
		if err == nil {
			fullText.WriteString(text)
			fullText.WriteString("\n\n")
		}
	}

	return fullText.String(), nil
}

func (f *ArxivFetcher) extractPDFText(pdfURL string) (string, error) {
	body, _, err := utils.GetWithRetry(pdfURL, nil)
	if err != nil {
		return "", err
	}

	r, err := pdf.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	_, err = buf.ReadFrom(b)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
