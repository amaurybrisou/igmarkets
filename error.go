package igmarkets

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func LightStreamErrorHandler(resp *http.Response, err error) error {
	if resp != nil {
		url := resp.Request.URL.String()
		body, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			return fmt.Errorf("calling lightstreamer endpoint %s failed: %v; reading HTTP body also failed: %v",
				url, err, err2)
		}
		return fmt.Errorf("calling lightstreamer endpoint %s failed: %v http.StatusCode:%d Body: %q",
			url, err, resp.StatusCode, string(body))
	}
	return fmt.Errorf("calling lightstreamer endpoint failed: %v", err)
}
