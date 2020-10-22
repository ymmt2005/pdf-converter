package sub

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

type hideR struct {
	r io.Reader
}

func (h hideR) Read(p []byte) (int, error) {
	return h.r.Read(p)
}

func makeRequest(method, url string, name, filename string, body []byte, chunked bool) *http.Request {
	if name == "not-multi" {
		req, err := http.NewRequest(method, url, strings.NewReader("dummy"))
		if err != nil {
			panic(err)
		}
		return req
	}

	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormField("ignored")
	if err != nil {
		panic(err)
	}
	fw.Write([]byte("value"))
	fw, err = w.CreateFormFile(name, filename)
	if err != nil {
		panic(err)
	}
	fw.Write(body)
	err = w.Close()
	if err != nil {
		panic(err)
	}
	var r io.Reader = buf
	if chunked {
		r = hideR{buf}
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestConvertHandler(t *testing.T) {
	cvt := mockConverter{false}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	s := httptest.NewServer(NewConvertHandler(cvt, dir, 1024, 1*time.Second, 0))
	defer s.Close()

	cases := []struct {
		name              string
		req               *http.Request
		expectCode        int
		expectContentType string
		expectBody        []byte
	}{
		{
			"bad-method",
			makeRequest(http.MethodGet, s.URL, "file", "t.pptx", []byte("dummy"), false),
			http.StatusMethodNotAllowed,
			"",
			nil,
		},
		{
			"no-content-length",
			makeRequest(http.MethodPost, s.URL, "file", "t.pptx", []byte("dummy"), true),
			http.StatusLengthRequired,
			"",
			nil,
		},
		{
			"too-large",
			makeRequest(http.MethodPost, s.URL, "file", "t.pptx", make([]byte, 2000), false),
			http.StatusRequestEntityTooLarge,
			"",
			nil,
		},
		{
			"not-multipart",
			makeRequest(http.MethodPost, s.URL, "not-multi", "t.pptx", []byte("dummy"), false),
			http.StatusBadRequest,
			"",
			nil,
		},
		{
			"no-file",
			makeRequest(http.MethodPost, s.URL, "wrong", "t.pptx", []byte("dummy"), false),
			http.StatusBadRequest,
			"",
			nil,
		},
		{
			"unsupported",
			makeRequest(http.MethodPost, s.URL, "file", "t.unsuported", []byte("dummy"), false),
			http.StatusUnsupportedMediaType,
			"",
			nil,
		},
		{
			"fail",
			makeRequest(http.MethodPost, s.URL, "file", "fail.pptx", []byte("dummy"), false),
			http.StatusInternalServerError,
			"",
			nil,
		},
		{
			"ok",
			makeRequest(http.MethodPost, s.URL, "file", "t.pptx", []byte("dummy"), false),
			http.StatusOK,
			"application/pdf",
			[]byte("dummy\n"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.Client().Do(tc.req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			if resp.StatusCode != tc.expectCode {
				t.Error(`resp.StatusCode != tc.expectCode`, resp.StatusCode)
			}
			if tc.expectCode != http.StatusOK {
				return
			}

			if resp.Header.Get("Content-Type") != tc.expectContentType {
				t.Error(`resp.Header.Get("Content-Type") != tc.expectContentType`, resp.Header.Get("Content-Type"))
			}
			if !bytes.Equal(body, tc.expectBody) {
				t.Error(`!bytes.Equal(body, tc.expectBody)`, string(body))
			}
		})
	}

	cvtHandler := NewConvertHandler(mockConverter{true}, dir, 1<<10, 1*time.Second, 2)
	ss := httptest.NewServer(cvtHandler)
	go ss.Client().Do(makeRequest(http.MethodPost, ss.URL, "file", "a.docx", []byte("foo"), false))
	go ss.Client().Do(makeRequest(http.MethodPost, ss.URL, "file", "a.docx", []byte("foo"), false))
	time.Sleep(10 * time.Millisecond)
	resp, err := ss.Client().Do(makeRequest(http.MethodPost, ss.URL, "file", "a.docx", []byte("foo"), false))
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Error(`resp.StatusCode != http.StatusTooManyRequests`, resp.StatusCode)
		t.Log(string(data))
	}
	ss.Close()

	// test that the working directory is empty after conversion
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, fi := range fis {
		if strings.HasPrefix(fi.Name(), "tmp") {
			t.Error("working directory is not empty:", fi.Name())
		}
	}

	// test metrics
	problems, err := promtest.GatherAndLint(prometheus.DefaultGatherer)
	if err != nil {
		t.Error(err, problems)
	}
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}

	m := findMetric(mfs, "pdf_converter_requests_total", map[string]string{"status": "200"})
	if m == nil {
		t.Fatal("no pdf_converter_requests_total[status=200]")
	}
	if int(*m.Counter.Value) != 3 {
		t.Error("pdf_converter_requests_total[status=200] != 3", int(*m.Counter.Value))
	}

	m = findMetric(mfs, "pdf_converter_requests_total", map[string]string{"status": "413"})
	if m == nil {
		t.Fatal("no pdf_converter_requests_total[status=413]")
	}
	if int(*m.Counter.Value) != 1 {
		t.Error("pdf_converter_requests_total[status=413] != 1", int(*m.Counter.Value))
	}

	m = findMetric(mfs, "pdf_converter_conversion_total", map[string]string{"extension": "pptx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_total[extension=pptx]")
	}
	if int(*m.Counter.Value) != 2 {
		t.Error("pdf_converter_conversion_total[extension=pptx] != 2", int(*m.Counter.Value))
	}

	m = findMetric(mfs, "pdf_converter_conversion_total", map[string]string{"extension": "docx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_total[extension=docx]")
	}
	if int(*m.Counter.Value) != 2 {
		t.Error("pdf_converter_conversion_total[extension=docx] != 2", int(*m.Counter.Value))
	}

	m = findMetric(mfs, "pdf_converter_conversion_failed", map[string]string{"extension": "pptx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_failed[extension=pptx]")
	}
	if int(*m.Counter.Value) != 1 {
		t.Error("pdf_converter_conversion_failed[extension=pptx] != 1", int(*m.Counter.Value))
	}

	m = findMetric(mfs, "pdf_converter_conversion_duration_seconds", map[string]string{"extension": "pptx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_duration_seconds[extension=pptx]")
	}
	if int(m.Histogram.GetSampleCount()) != 2 {
		t.Error("pdf_converter_conversion_duration_seconds_count[extension=pptx] != 2", int(m.Histogram.GetSampleCount()))
	}

	m = findMetric(mfs, "pdf_converter_conversion_duration_seconds", map[string]string{"extension": "docx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_duration_seconds[extension=docx]")
	}
	if m.Histogram.GetSampleSum() < 2 {
		t.Error("pdf_converter_conversion_duration_seconds_sum[extension=docx] < 2.0", m.Histogram.GetSampleSum())
	}

	m = findMetric(mfs, "pdf_converter_conversion_source_bytes", map[string]string{"extension": "docx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_source_bytes[extension=docx]")
	}
	if int(m.Histogram.GetSampleSum()) != 6 {
		t.Error("pdf_converter_conversion_source_bytes_sum[extension=docx] != 6", int(m.Histogram.GetSampleSum()))
	}

	m = findMetric(mfs, "pdf_converter_conversion_output_bytes", map[string]string{"extension": "docx"})
	if m == nil {
		t.Fatal("no pdf_converter_conversion_output_bytes[extension=docx]")
	}
	if int(m.Histogram.GetSampleSum()) != 12 { // 2 * sizeof(testdata/converted.pdf)
		t.Error("wrong pdf_converter_conversion_output_bytes_sum[extension=docx]", int(m.Histogram.GetSampleSum()))
	}
}

type mockConverter struct {
	wait bool
}

func (c mockConverter) Supported(filename string) bool {
	switch filepath.Ext(filename) {
	case ".pptx", ".docx":
		return true
	}
	return false
}

func (c mockConverter) Convert(ctx context.Context, filePath string) (convertedPath string, err error) {
	if filepath.Base(filePath) == "fail.pptx" {
		return "", errors.New("fail")
	}
	if c.wait {
		<-ctx.Done()
	}
	return "testdata/converted.pdf", nil
}

func findMetric(mfs []*dto.MetricFamily, name string, labels map[string]string) *dto.Metric {
	for _, mf := range mfs {
		if mf.Name == nil {
			continue
		}
		if *mf.Name != name {
			continue
		}
		for _, m := range mf.Metric {
			if hasLabels(m.Label, labels) {
				return m
			}
		}
	}
	return nil
}

func hasLabels(pairs []*dto.LabelPair, labels map[string]string) bool {
	t := make(map[string]string)
	for _, pair := range pairs {
		t[pair.GetName()] = pair.GetValue()
	}

	for k, v := range labels {
		if t[k] != v {
			return false
		}
	}

	return true
}
