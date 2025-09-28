package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type RequestOptions[T any] struct {
	Body        io.Reader
	ContentType string
	Headers     map[string]string
	QueryParams map[string]string
	Result      *T
	Debug       bool
}

func NewRequestOptions[T any](contentType string, result *T) *RequestOptions[T] {
	headers := make(map[string]string)
	headers["Content-Type"] = contentType

	return &RequestOptions[T]{
		Body:        nil,
		ContentType: contentType,
		Headers:     headers,
		Result:      result,
		Debug:       false,
	}
}

func (o *RequestOptions[T]) AddHeader(key string, value string) {
	o.Headers[key] = value
}

func (o *RequestOptions[T]) AddQueryParam(key string, value string) {
	o.QueryParams[key] = value
}

func (o *RequestOptions[T]) SetBody(body any) {
	if o.ContentType == "application/json" {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return
		}
		o.Body = bytes.NewBuffer(jsonBody)
		return
	}

	if o.ContentType == "application/x-www-form-urlencoded" {
		// Read all properties of the body
		values := url.Values{}
		for key, value := range body.(map[string]string) {
			values.Add(key, value)
		}
		o.Body = strings.NewReader(values.Encode())
		return
	}

	log.Fatal("unsupported content type: ", o.ContentType)
}

func DoRequest[T any](method string, uri string, options *RequestOptions[T]) ([]byte, error) {
	httpClient := &http.Client{}

	if options != nil && len(options.QueryParams) > 0 {
		queryParams := url.Values{}
		for key, value := range options.QueryParams {
			queryParams.Add(key, value)
		}
		uri += "?" + queryParams.Encode()
	}

	body := io.Reader(nil)
	if options != nil {
		body = options.Body
	}

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	if options != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	//fmt.Println("[*] Request:", req.URL.String())
	//fmt.Println("[*] Request Headers:", req.Header)
	//fmt.Println("[*] Request Body:", body)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		if resp.Header.Get("Content-Type") == "application/json" {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			var errorResponse map[string]string
			err = json.Unmarshal(body, &errorResponse)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("status code: %d, body: %s", resp.StatusCode, errorResponse["error_description"])
		} else {
			return nil, fmt.Errorf("status code: %d", resp.StatusCode)
		}
	}

	bytesBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if options != nil && options.Result != nil {
		err = json.Unmarshal(bytesBody, options.Result)
		return nil, err
	}

	return bytesBody, nil
}
