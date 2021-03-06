//
// Copyright (C) 2018 - 2020 assay.it
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/assay-it/sdk-go
//

package recv_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/assay-it/sdk-go/assay"
	µ "github.com/assay-it/sdk-go/http"
	ƒ "github.com/assay-it/sdk-go/http/recv"
	ø "github.com/assay-it/sdk-go/http/send"
)

func TestCodeOk(t *testing.T) {
	ts := mock()
	defer ts.Close()

	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil {
		t.Error("fail to handle request")
	}
}

func TestCodeNoMatch(t *testing.T) {
	ts := mock()
	defer ts.Close()

	req := µ.Join(
		ø.GET(ts.URL+"/other"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
	)
	cat := req(µ.DefaultIO())

	if !errors.Is(cat.Fail, µ.StatusBadRequest) {
		t.Error("fail to detect code mismatch")
	}
}

func TestHeaderOk(t *testing.T) {
	ts := mock()
	defer ts.Close()

	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
		ƒ.Header("content-type").Is("application/json"),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil {
		t.Error("fail to match header value")
	}
}

func TestHeaderAny(t *testing.T) {
	ts := mock()
	defer ts.Close()

	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
		ƒ.Header("content-type").Any(),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil {
		t.Error("fail to match header value")
	}
}

func TestHeaderVal(t *testing.T) {
	ts := mock()
	defer ts.Close()

	var content string
	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
		ƒ.Header("content-type").String(&content),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil || content != "application/json" {
		t.Error("fail to match header value")
	}
}

func TestHeaderMismatch(t *testing.T) {
	ts := mock()
	defer ts.Close()

	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
		ƒ.Header("content-type").Is("foo/bar"),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail == nil {
		t.Error("fail to detect header mismatch")
	}
}

func TestHeaderUndefinedWithLit(t *testing.T) {
	ts := mock()
	defer ts.Close()

	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
		ƒ.Header("x-content-type").Is("foo/bar"),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail == nil {
		t.Error("fail to detect missing header")
	}
}

func TestHeaderUndefinedWithVal(t *testing.T) {
	ts := mock()
	defer ts.Close()

	var val string
	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ø.AcceptJSON(),
		ƒ.Code(µ.StatusOK),
		ƒ.Header("x-content-type").String(&val),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail == nil {
		t.Error("fail to detect missing header")
	}
}

func TestRecvJSON(t *testing.T) {
	type Site struct {
		Site string `json:"site"`
	}

	ts := mock()
	defer ts.Close()

	var site Site
	req := µ.Join(
		ø.GET(ts.URL+"/json"),
		ƒ.Code(µ.StatusOK),
		ƒ.ServedJSON(),
		ƒ.Recv(&site),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil || site.Site != "example.com" {
		t.Error("failed to receive json")
	}
}

func TestRecvForm(t *testing.T) {
	type Site struct {
		Site string `json:"site"`
	}

	ts := mock()
	defer ts.Close()

	var site Site
	req := µ.Join(
		ø.GET(ts.URL+"/form"),
		ƒ.Code(µ.StatusOK),
		ƒ.ServedForm(),
		ƒ.Recv(&site),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil || site.Site != "example.com" {
		t.Error("failed to receive json")
	}
}

func TestRecvBytes(t *testing.T) {
	type Site struct {
		Site string `json:"site"`
	}

	ts := mock()
	defer ts.Close()

	var data []byte
	req := µ.Join(
		ø.GET(ts.URL+"/form"),
		ƒ.Code(µ.StatusOK),
		ƒ.Served().Any(),
		ƒ.Bytes(&data),
	)
	cat := assay.IO(µ.Default())

	if cat = req(cat); cat.Fail != nil || string(data) != "site=example.com" {
		t.Error("failed to receive json")
	}
}

//
func mock() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/json":
				w.Header().Add("Content-Type", "application/json")
				w.Write([]byte(`{"site": "example.com"}`))
			case r.URL.Path == "/form":
				w.Header().Add("Content-Type", "application/x-www-form-urlencoded")
				w.Write([]byte("site=example.com"))
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
		}),
	)
}
