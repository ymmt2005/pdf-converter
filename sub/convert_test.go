package sub

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
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
	defer ss.Close()
	go ss.Client().Do(makeRequest(http.MethodPost, ss.URL, "file", "a.pptx", []byte("foo"), false))
	go ss.Client().Do(makeRequest(http.MethodPost, ss.URL, "file", "a.pptx", []byte("foo"), false))
	time.Sleep(10 * time.Millisecond)
	resp, err := ss.Client().Do(makeRequest(http.MethodPost, ss.URL, "file", "a.pptx", []byte("foo"), false))
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
}

type mockConverter struct {
	wait bool
}

func (c mockConverter) Supported(filename string) bool {
	return strings.HasSuffix(filename, ".pptx")
}

func (c mockConverter) Convert(ctx context.Context, filePath string) (convertedPath string, err error) {
	if c.wait {
		<-ctx.Done()
	}
	return "testdata/converted.pdf", nil
}
