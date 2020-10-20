package sub

import (
	"net/http"
	"time"
)

func NewPDFConverter(dir string, maxLength int64, maxConvertTime time.Duration, maxParallel int) http.Handler {
	var pCh chan struct{}
	if maxParallel > 0 {
		pCh = make(chan struct{}, maxParallel)
	}
	return &pdfConverter{dir, maxLength, maxConvertTime, pCh}
}

type pdfConverter struct {
	dir         string
	maxLength   int64
	maxDuration time.Duration
	parallelism <-chan struct{}
}

func (c *pdfConverter) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}
