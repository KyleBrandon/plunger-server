package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequest(t *testing.T, method string, url string, body io.Reader, handler func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler(w, req)

	return w
}

func TestRequestWithHeaders(t *testing.T, method string, url string, headers map[string][]string, body io.Reader, handler func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range headers {
		for _, h := range v {
			req.Header.Add(k, h)
		}
	}

	w := httptest.NewRecorder()
	handler(w, req)

	return w
}
