package sub

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ymmt2005/pdf-converter/converter"
)

// NewConvertHandler creates an http.Handler for /convert API
func NewConvertHandler(cvt converter.Converter, dir string, maxLength int64, maxConvertTime time.Duration, maxParallel int) http.Handler {
	var pCh chan struct{}
	if maxParallel > 0 {
		pCh = make(chan struct{}, maxParallel)
	}
	return &pdfConverter{cvt, dir, maxLength, maxConvertTime, pCh}
}

type pdfConverter struct {
	cvt         converter.Converter
	dir         string
	maxLength   int64
	maxDuration time.Duration
	parallelism chan struct{}
}

func (c *pdfConverter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.parallelism != nil {
		select {
		case c.parallelism <- struct{}{}:
			defer func() {
				<-c.parallelism
			}()
		default:
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
	}

	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}
	lengthStr := r.Header.Get("Content-Length")
	if len(lengthStr) == 0 {
		http.Error(w, "content-length header is required", http.StatusLengthRequired)
		return
	}
	length, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad length: %s: %v", lengthStr, err), http.StatusBadRequest)
		return
	}
	if length > c.maxLength {
		http.Error(w, fmt.Sprintf("the file size exceeds the limit %d", c.maxLength), http.StatusRequestEntityTooLarge)
		return
	}

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, fmt.Sprintf("the body is not multipart: %v", err), http.StatusBadRequest)
		return
	}

	var filePart *multipart.Part
	for p, err := mr.NextPart(); err != io.EOF; p, err = mr.NextPart() {
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if p.FormName() == "file" {
			filePart = p
			break
		}
		p.Close()
	}

	if filePart == nil {
		http.Error(w, "no file", http.StatusBadRequest)
		return
	}
	defer filePart.Close()

	filename := filepath.Base(filePart.FileName())
	if len(filename) == 0 {
		http.Error(w, "no filename", http.StatusBadRequest)
		return
	}

	if !c.cvt.Supported(filename) {
		http.Error(w, "unsupported file type", http.StatusUnsupportedMediaType)
		return
	}

	tmpdir, err := ioutil.TempDir(c.dir, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpdir)

	filePath := filepath.Join(c.dir, filename)
	f, err := os.Create(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, filePart); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), c.maxDuration)
	defer cancel()
	converted, err := c.cvt.Convert(ctx, filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	convertedR, err := os.Open(converted)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer convertedR.Close()

	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, convertedR)
}
