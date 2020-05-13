package parser

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

var errBadResponseCode = errors.New("bad response code")

func fromURL(url string) (body []byte, err error) {
	client := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: %w", err)
	}

	req.Header.Set("User-Agent", "Mynews/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request http request: %w", err)
	}

	if resp != nil {
		defer func() {
			ce := resp.Body.Close()
			if ce != nil {
				err = ce
			}
		}()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errBadResponseCode
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return body, nil
}