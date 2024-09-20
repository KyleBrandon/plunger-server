package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestExpectedStatus(t *testing.T, rr *httptest.ResponseRecorder, statusCode int) {
	if rr.Code != statusCode {
		t.Errorf("expected status code %d, got %d", statusCode, rr.Code)
	}
}

func TestExpectedMessage(t *testing.T, rr *httptest.ResponseRecorder, m string) {
	if !strings.Contains(rr.Body.String(), m) {
		t.Errorf("received error message `%s`, expected message `%s`", rr.Body.String(), m)
	}
}
