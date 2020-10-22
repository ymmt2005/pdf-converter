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
	"strings"
	"time"

	"github.com/cybozu-go/log"
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

func sendError(w http.ResponseWriter, msg string, code int) {
	requestsTotal.WithLabelValues(strconv.Itoa(code)).Inc()
	http.Error(w, msg, code)
}

func (c *pdfConverter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.parallelism != nil {
		select {
		case c.parallelism <- struct{}{}:
			defer func() {
				<-c.parallelism
			}()
		default:
			sendError(w, "too many requests", http.StatusTooManyRequests)
			return
		}
	}

	if r.Method != http.MethodPost {
		sendError(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}
	lengthStr := r.Header.Get("Content-Length")
	if len(lengthStr) == 0 {
		sendError(w, "content-length header is required", http.StatusLengthRequired)
		return
	}
	length, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil {
		sendError(w, fmt.Sprintf("bad length: %s: %v", lengthStr, err), http.StatusBadRequest)
		return
	}
	if length > c.maxLength {
		sendError(w, fmt.Sprintf("the file size exceeds the limit %d", c.maxLength), http.StatusRequestEntityTooLarge)
		return
	}

	mr, err := r.MultipartReader()
	if err != nil {
		sendError(w, fmt.Sprintf("the body is not multipart: %v", err), http.StatusBadRequest)
		return
	}

	var filePart *multipart.Part
	for p, err := mr.NextPart(); err != io.EOF; p, err = mr.NextPart() {
		if err != nil {
			sendError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if p.FormName() == "file" {
			filePart = p
			break
		}
		p.Close()
	}

	if filePart == nil {
		sendError(w, "no file", http.StatusBadRequest)
		return
	}
	defer filePart.Close()

	filename := filepath.Base(string([]rune(filePart.FileName())))
	if len(filename) == 0 {
		sendError(w, "no filename", http.StatusBadRequest)
		return
	}

	if !c.cvt.Supported(filename) {
		sendError(w, "unsupported file type", http.StatusUnsupportedMediaType)
		return
	}

	extension := strings.ToLower(filepath.Ext(filename))
	if len(extension) > 0 {
		extension = extension[1:]
	}

	tmpdir, err := ioutil.TempDir(c.dir, "tmp")
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpdir)

	filePath := filepath.Join(tmpdir, filename)
	f, err := os.Create(filePath)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	srcLen, err := io.Copy(f, filePart)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	begin := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), c.maxDuration)
	defer cancel()
	converted, err := c.cvt.Convert(ctx, filePath)

	conversionTotal.WithLabelValues(extension).Inc()
	conversionSeconds.WithLabelValues(extension).Observe(time.Since(begin).Seconds())
	if err != nil {
		conversionFailed.WithLabelValues(extension).Inc()
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	convertedR, err := os.Open(converted)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer convertedR.Close()

	w.Header().Set("Content-Type", "application/pdf")
	outputLen, err := io.Copy(w, convertedR)
	if err != nil {
		log.Warn("failed to send PDF body", map[string]interface{}{
			"filename":  filename,
			log.FnError: err,
		})
	}

	requestsTotal.WithLabelValues(strconv.Itoa(http.StatusOK)).Inc()
	sourceBytes.WithLabelValues(extension).Observe(float64(srcLen))
	outputBytes.WithLabelValues(extension).Observe(float64(outputLen))
}
