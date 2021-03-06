//
// Copyright (C) 2018 - 2020 assay.it
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/assay-it/sdk-go
//

package send

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/assay-it/sdk-go/assay"
	"github.com/assay-it/sdk-go/http"
)

//-------------------------------------------------------------------
//
// core arrows
//
//-------------------------------------------------------------------

/*
URL defines a mandatory parameters to the request such as
HTTP method and destination URL, use Params arrow if you
need to supply URL query params.
*/
func URL(method, uri string, args ...interface{}) http.Arrow {
	return func(cat *assay.IOCat) *assay.IOCat {
		var addr *url.URL
		if addr, cat.Fail = mkURL(uri, args...); cat.Fail != nil {
			return cat
		}

		if cat.HTTP == nil {
			cat.HTTP = &assay.IOCatHTTP{}
		}

		switch addr.Scheme {
		case "http", "https":
			cat.HTTP.Send = &assay.UpStreamHTTP{
				Method:  method,
				URL:     addr,
				Header:  make(map[string]*string),
				Payload: bytes.NewBuffer(nil),
			}
		default:
			cat.Fail = &assay.NotSupported{URL: addr}
		}
		return cat
	}
}

func mkURL(uri string, args ...interface{}) (*url.URL, error) {
	switch uri[0] {
	case '!':
		return mkEscapedURL(false, uri[1:], args...)
	default:
		return mkEscapedURL(true, uri, args...)
	}
}

func mkEscapedURL(escape bool, uri string, args ...interface{}) (*url.URL, error) {
	opts := []interface{}{}
	for _, x := range args {
		switch v := x.(type) {
		case *url.URL:
			v.Path = strings.TrimSuffix(v.Path, "/")
			opts = append(opts, v.String())
		case func() string:
			opts = append(opts, maybeEscape(escape, v()))
		default:
			opts = append(opts, maybeEscape(escape, urlSegment(x)))
		}
	}

	return url.Parse(fmt.Sprintf(uri, opts...))
}

func urlSegment(arg interface{}) string {
	val := reflect.ValueOf(arg)

	if val.Kind() == reflect.Ptr {
		return fmt.Sprintf("%v", val.Elem())
	}

	return fmt.Sprintf("%v", val)
}

func maybeEscape(escape bool, val string) string {
	if escape {
		return url.PathEscape(val)
	}

	return val
}

/*

THeader is tagged string, represents HTTP Header
*/
type THeader struct{ string }

/*

Header defines HTTP headers to the request, use combinator
to define multiple header values.

  http.HTTP(
		ø.Header("Accept").Is(...),
		ø.Header("Content-Type").Is(...),
	)
*/
func Header(header string) THeader {
	return THeader{header}
}

func (header THeader) name() string {
	return strings.ToLower(header.string)
}

// Is sets a literval value of HTTP header
func (header THeader) Is(value string) http.Arrow {
	return func(cat *assay.IOCat) *assay.IOCat {
		cat.HTTP.Send.Header[header.name()] = &value
		return cat
	}
}

// Val sets a value of HTTP header from variable
func (header THeader) Val(value *string) http.Arrow {
	return func(cat *assay.IOCat) *assay.IOCat {
		cat.HTTP.Send.Header[header.name()] = value
		return cat
	}
}

/*

Params appends query params to request URL. The arrow takes a struct and
converts it to map[string]string. The function fails if input is not convertable
to map of strings (e.g. nested struct).
*/
func Params(query interface{}) http.Arrow {
	return func(cat *assay.IOCat) *assay.IOCat {
		bytes, err := json.Marshal(query)
		if err != nil {
			cat.Fail = err
			return cat
		}

		var req map[string]string
		err = json.Unmarshal(bytes, &req)
		if err != nil {
			cat.Fail = err
			return cat
		}

		q := cat.HTTP.Send.URL.Query()
		for k, v := range req {
			q.Add(k, v)
		}
		cat.HTTP.Send.URL.RawQuery = q.Encode()
		return cat
	}
}

/*

Send payload to destination URL. You can also use native Go data types
(e.g. maps, struct, etc) as egress payload. The library implicitly encodes
input structures to binary using Content-Type as a hint. The function fails
if content type is not supported by the library.
*/
func Send(data interface{}) http.Arrow {
	return func(cat *assay.IOCat) *assay.IOCat {
		content, ok := cat.HTTP.Send.Header["content-type"]
		if !ok {
			cat.Fail = fmt.Errorf("unknown Content-Type")
			return cat
		}

		switch stream := data.(type) {
		case string:
			cat.HTTP.Send.Payload = bytes.NewBuffer([]byte(stream))
		case []byte:
			cat.HTTP.Send.Payload = bytes.NewBuffer(stream)
		case io.Reader:
			cat.HTTP.Send.Payload = stream
		default:
			cat.HTTP.Send.Payload, cat.Fail = encode(*content, data)
		}
		return cat
	}
}

func encode(content string, data interface{}) (buf *bytes.Buffer, err error) {
	switch {
	// "application/json" and other variants
	case strings.Contains(content, "json"):
		buf, err = encodeJSON(data)
	// "application/x-www-form-urlencoded"
	case strings.Contains(content, "www-form"):
		buf, err = encodeForm(data)
	default:
		err = fmt.Errorf("unsupported Content-Type %v", content)
	}

	return
}

func encodeJSON(data interface{}) (*bytes.Buffer, error) {
	json, err := json.Marshal(data)
	return bytes.NewBuffer(json), err
}

func encodeForm(data interface{}) (*bytes.Buffer, error) {
	bin, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var req map[string]string
	err = json.Unmarshal(bin, &req)
	if err != nil {
		return nil, fmt.Errorf("encode application/x-www-form-urlencoded: %w", err)
	}

	var payload url.Values = make(map[string][]string)
	for key, val := range req {
		payload[key] = []string{val}
	}
	return bytes.NewBuffer([]byte(payload.Encode())), nil
}

//-------------------------------------------------------------------
//
// arrows aliases
//
//-------------------------------------------------------------------

// GET is syntax sugar of URL("GET", ...)
func GET(uri string, args ...interface{}) http.Arrow {
	return URL("GET", uri, args...)
}

// POST is syntax sugar of URL("POST", ...)
func POST(uri string, args ...interface{}) http.Arrow {
	return URL("POST", uri, args...)
}

// PUT is syntax sugar of URL("PUT", ...)
func PUT(uri string, args ...interface{}) http.Arrow {
	return URL("PUT", uri, args...)
}

// DELETE is syntax sugar of URL("DELETE", ...)
func DELETE(uri string, args ...interface{}) http.Arrow {
	return URL("DELETE", uri, args...)
}

// Accept is syntax sugar of Header("Accept")
func Accept() THeader {
	return Header("Accept")
}

// AcceptJSON is syntax sugar of Header("Accept").Is("application/json")
func AcceptJSON() http.Arrow {
	return Accept().Is("application/json")
}

// AcceptForm is syntax sugar of Header("Accept").Is("application/x-www-form-urlencoded")
func AcceptForm() http.Arrow {
	return Accept().Is("application/x-www-form-urlencoded")
}

// Content is syntax sugar of Header("Content-Type")
func Content() THeader {
	return Header("Content-Type")
}

// ContentJSON is syntax sugar of Header("Content-Type").Is("application/json")
func ContentJSON() http.Arrow {
	return Content().Is("application/json")
}

// ContentForm is syntax sugar of Header("Content-Type").Is("application/x-www-form-urlencoded")
func ContentForm() http.Arrow {
	return Content().Is("application/x-www-form-urlencoded")
}

// KeepAlive is a syntax sugar of Header("Connection").Is("keep-alive")
func KeepAlive() http.Arrow {
	return Header("Connection").Is("keep-alive")
}

// Authorization is syntax sugar of Header("Authorization")
func Authorization() THeader {
	return Header("Authorization")
}
