package igmarkets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func readLightStreamSubscription(epics, fields []string, tickReceiver chan LightStreamChartTick, resp *http.Response) {
	var respBuf = make([]byte, 64)
	var lastTicks = make(map[string]LightStreamChartTick, len(epics)) // epic -> tick

	defer close(tickReceiver)

	// map table index -> epic name
	var epicIndex = make(map[string]string, len(epics))
	for i, epic := range epics {
		epicIndex[fmt.Sprintf("1,%d", i+1)] = epic
	}

	for {
		read, err := resp.Body.Read(respBuf)

		// fmt.Println(string(respBuf[0:read]), err)
		if read > 0 {
			mess := string(respBuf[0:read])
			if mess == "LOOP\r\n\r\n" {
				fmt.Printf("Server Closed Stream\n")
				break
			}
		} // Sever ends streaming

		if err != nil {
			if err == io.EOF {
				fmt.Printf("Server Closed Stream\n")
				break
			}
			fmt.Printf("reading lightstreamer subscription failed: %v", err)
			break
		}

		priceMsg := string(respBuf[0:read])
		priceParts := strings.Split(priceMsg, "|")

		if len(priceParts) != len(fields)+1 {
			continue
		}

		epic, ok := epicIndex[priceParts[0]]
		if !ok {
			continue
		}

		tick, err := NewLightStreamChartTick(epic, fields, priceParts[1:])

		tick.Merge(lastTicks[epic])

		if err != nil {
			fmt.Printf("lighstream could not parse tick %v", err)
			continue
		}

		tickReceiver <- tick
		lastTicks[epic] = tick
	}
}

// LoginVersion2 - use old login version. contains required data for LightStreamer API
func (ig *IGMarkets) LoginVersion2() (*SessionVersion2, error) {
	bodyReq := new(bytes.Buffer)

	var authReq = authRequest{
		Identifier: ig.Identifier,
		Password:   ig.Password,
	}

	if err := json.NewEncoder(bodyReq).Encode(authReq); err != nil {
		return nil, fmt.Errorf("igmarkets: unable to encode JSON response: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", ig.APIURL, "gateway/deal/session"), bodyReq)
	if err != nil {
		return nil, fmt.Errorf("igmarkets: unable to send HTTP request: %v", err)
	}

	igResponseInterface, headers, err := ig.doRequestWithResponseHeaders(req, 2, SessionVersion2{}, false)
	if err != nil {
		return nil, err
	}
	session, _ := igResponseInterface.(*SessionVersion2)
	if headers != nil {
		session.CSTToken = headers.Get("CST")
		session.XSTToken = headers.Get("X-SECURITY-TOKEN")
	}
	return session, nil
}
