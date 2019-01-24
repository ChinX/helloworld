package restful

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func NewRequest(method string, addr string, header http.Header, body interface{}) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		switch v := body.(type) {
		case io.Reader:
			r = v
		case string:
			r = strings.NewReader(v)
		case []byte:
			r = bytes.NewReader(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf(" marshal request body faild: %s", err)
			}
			r = bytes.NewReader(data)
		}
	}
	req, err := http.NewRequest(method, addr, r)
	if err != nil {
		return nil, err
	}

	if header != nil {
		req.Header = header
	}
	return req, nil
}

func DoRequest(req *http.Request, expectation interface{}) error {
	client := http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request faild: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response faild: %s", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		if expectation != nil{
			log.Println(string(body))
			err := json.Unmarshal(body, expectation)
			if err != nil {
				return fmt.Errorf("unmarshal response body: \"%s\" faild: %s", string(body), err)
			}
		}
		return nil
	}
	return fmt.Errorf("do request failed, response statusCode: %d, body: %s",
		resp.StatusCode, string(body))
}
