package dmcbulletins

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// pdfTextTimeout bounds one PDF's extraction.
const pdfTextTimeout = 30 * time.Second

// maxPDFText caps the text kept from one PDF.
const maxPDFText = 1 << 20

// PDFToText extracts text with poppler's pdftotext, which the production
// image installs. Its output, joined into paragraphs, reads as the index's
// own text does.
type PDFToText struct {
	path string
}

// NewPDFToText finds pdftotext on the PATH, or returns nil when it is not
// installed, which leaves bulletin text to the index.
func NewPDFToText() *PDFToText {
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		return nil
	}
	return &PDFToText{path: path}
}

// Text implements TextExtractor.
func (extractor *PDFToText) Text(ctx context.Context, pdf []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, pdfTextTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, extractor.path, "-q", "-enc", "UTF-8", "-", "-") //nolint:gosec // a fixed binary and arguments
	command.Stdin = bytes.NewReader(pdf)
	var out, stderr bytes.Buffer
	command.Stdout, command.Stderr = &limited{buffer: &out, left: maxPDFText}, &limited{buffer: &stderr, left: 4096}
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("pdftotext timed out")
		}
		return "", fmt.Errorf("pdftotext: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out.String(), nil
}

// limited keeps at most left bytes and drops the rest.
type limited struct {
	buffer *bytes.Buffer
	left   int
}

func (writer *limited) Write(data []byte) (int, error) {
	if writer.left > 0 {
		keep := data
		if len(keep) > writer.left {
			keep = keep[:writer.left]
		}
		writer.buffer.Write(keep)
		writer.left -= len(keep)
	}
	return len(data), nil
}
