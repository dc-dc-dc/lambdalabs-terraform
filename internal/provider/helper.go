package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func MakeAPICall(ctx context.Context, apiKey, method, url string, data interface{}) (*http.Response, error) {
	var reader io.Reader
	if data != nil {
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("https://cloud.lambdalabs.com/api/v1/%s", url), reader)
	httpReq.SetBasicAuth(apiKey, "")
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(httpReq)

}
