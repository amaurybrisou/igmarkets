package igmarkets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func readLightStreamSubscription(epics, fields []string, tickReceiver chan LightStreamChartTick, body io.ReadCloser, errChan chan error) {
	var respBuf = make([]byte, 64)
	var lastTicks = make(map[string]LightStreamChartTick, len(epics)) // epic -> tick

	defer close(tickReceiver)
	defer close(errChan)
	defer body.Close()

	// map table index -> epic name
	var epicIndex = make(map[string]string, len(epics))
	for i, epic := range epics {
		epicIndex[fmt.Sprintf("1,%d", i+1)] = epic
	}

	log.Debug("lightstreamer : reading stream")

	for {
		read, err := body.Read(respBuf)

		log.Traceln(string(respBuf[0:read]), err)

		if read > 0 {
			mess := string(respBuf[:read])
			if mess == "LOOP\r\n\r\n" {
				errChan <- fmt.Errorf("recv LOOP")
				log.Printf("Server Closed Stream\n")

				return
			}
		} // Sever ends streaming

		if err != nil {
			if err == io.EOF {
				log.Printf("Server Closed Stream\n")
				errChan <- fmt.Errorf("recv EOF")
				return
			}
			errChan <- err
			log.Printf("reading lightstreamer subscription failed: %v", err)
			return
		}

		priceMsg := string(respBuf[:read])
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
			log.Printf("lighstream could not parse tick %v", err)
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
