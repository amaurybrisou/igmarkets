package igmarkets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type LightStreamerTick struct {
	Epic       string
	Time       time.Time
	OpenPrice  float64
	MaxPrice   float64
	MinPrice   float64
	ClosePrice float64
}

func (ig *IGMarkets) CloseLightStreamerSubscription() error {
	const contentType = "application/x-www-form-urlencoded"

	tr := &http.Transport{
		MaxIdleConns:       1,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	c := &http.Client{Transport: tr}

	body := []byte("LS_table=1&LS_op=destroy&LS_session=" + ig.SessionID)
	bodyBuf := bytes.NewBuffer(body)
	url := fmt.Sprintf("%s/lightstreamer/control.txt", ig.SessionVersion2.LightstreamerEndpoint)
	resp, err := c.Post(url, contentType, bodyBuf)
	if err != nil {
		if resp != nil {
			body, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				return fmt.Errorf("calling lightstreamer endpoint %s failed: %v; reading HTTP body also failed: %v",
					url, err, err2)
			}
			return fmt.Errorf("calling lightstreamer endpoint %q failed: %v http.StatusCode:%d Body: %q",
				url, err, resp.StatusCode, string(body))
		}
		return fmt.Errorf("calling lightstreamer endpoint %q failed: %v", url, err)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if body == "SYNC ERROR" {
		return fmt.Errorf("SYNC ERROR")
	}

	fmt.Printf("Unsubscription success to %s", string(body))

	return nil

}

// GetOTCWorkingOrders - Get all working orders
// epic: e.g. CS.D.BITCOIN.CFD.IP
// tickReceiver: receives all ticks from lightstreamer API
func (ig *IGMarkets) OpenLightStreamerSubscription(epics, fields []string, subType, tick, mode string, tickReceiver chan LightStreamerTick) error {
	const contentType = "application/x-www-form-urlencoded"

	// Obtain CST and XST tokens first
	sessionVersion2, err := ig.LoginVersion2()
	if err != nil {
		return fmt.Errorf("ig.LoginVersion2() failed: %v", err)
	}

	ig.SessionVersion2 = *sessionVersion2

	tr := &http.Transport{
		MaxIdleConns:       1,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	c := &http.Client{Transport: tr}

	// Create Lightstreamer Session
	body := []byte("LS_polling=true&LS_polling_millis=0&LS_idle_millis=0&LS_op2=create&LS_password=CST-" +
		sessionVersion2.CSTToken + "|" + "XST-" + sessionVersion2.XSTToken + "&LS_user=" +
		sessionVersion2.CurrentAccountId + "&LS_cid=mgQkwtwdysogQz2BJ4Ji kOj2Bg")
	bodyBuf := bytes.NewBuffer(body)
	url := fmt.Sprintf("%s/lightstreamer/create_session.txt", sessionVersion2.LightstreamerEndpoint)
	resp, err := c.Post(url, contentType, bodyBuf)
	if err != nil {
		if resp != nil {
			body, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				return fmt.Errorf("calling lightstreamer endpoint %s failed: %v; reading HTTP body also failed: %v",
					url, err, err2)
			}
			return fmt.Errorf("calling lightstreamer endpoint %s failed: %v http.StatusCode:%d Body: %q",
				url, err, resp.StatusCode, string(body))
		}
		return fmt.Errorf("calling lightstreamer endpoint %q failed: %v", url, err)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	sessionMsg := string(respBody[:])
	if !strings.HasPrefix(sessionMsg, "OK") {
		return fmt.Errorf("unexpected response from lightstreamer session endpoint %q: %q", url, sessionMsg)
	}
	sessionParts := strings.Split(sessionMsg, "\r\n")
	sessionID := sessionParts[1]
	sessionID = strings.ReplaceAll(sessionID, "SessionId:", "")
	ig.SessionID = sessionID

	// Adding subscription for epic
	var epicList string
	for i := range epics {
		epicList = epicList + subType + ":" + epics[i] + ":" + tick + "+"
	}

	body = []byte("LS_session=" + sessionID +
		"&LS_polling=true&LS_polling_millis=0&LS_idle_millis=0&LS_op=add&LS_Table=1&LS_id=" +
		epicList + "&LS_schema=" + strings.Join(fields[:], "+") + "&LS_mode=" + mode)
	bodyBuf = bytes.NewBuffer(body)
	url = fmt.Sprintf("%s/lightstreamer/control.txt", sessionVersion2.LightstreamerEndpoint)
	resp, err = c.Post(url, contentType, bodyBuf)
	if err != nil {
		if resp != nil {
			body, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				return fmt.Errorf("calling lightstreamer endpoint %s failed: %v; reading HTTP body also failed: %v",
					url, err, err2)
			}
			return fmt.Errorf("calling lightstreamer endpoint %q failed: %v http.StatusCode:%d Body: %q",
				url, err, resp.StatusCode, string(body))
		}
		return fmt.Errorf("calling lightstreamer endpoint %q failed: %v", url, err)
	}
	body, _ = ioutil.ReadAll(resp.Body)
	if !strings.HasPrefix(sessionMsg, "OK") {
		return fmt.Errorf("unexpected control.txt response: %q", body)
	}

	// Binding to subscription
	body = []byte("LS_session=" + sessionID + "&LS_polling=false&LS_polling_millis=0&LS_idle_millis=0")
	bodyBuf = bytes.NewBuffer(body)
	url = fmt.Sprintf("%s/lightstreamer/bind_session.txt", sessionVersion2.LightstreamerEndpoint)
	resp, err = c.Post(url, contentType, bodyBuf)
	if err != nil {
		if resp != nil {
			body, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				return fmt.Errorf("calling lightstreamer endpoint %s failed: %v; reading HTTP body also failed: %v",
					url, err, err2)
			}
			return fmt.Errorf("calling lightstreamer endpoint %q failed: %v http.StatusCode:%d Body: %q",
				url, err, resp.StatusCode, string(body))
		}
		return fmt.Errorf("calling lightstreamer endpoint %q failed: %v", url, err)
	}
	go readLightStreamSubscription(epics, fields, tickReceiver, resp)
	return nil
}

func readLightStreamSubscription(epics, fields []string, tickReceiver chan LightStreamerTick, resp *http.Response) {
	const epicNameUnknown = "unkown"
	var respBuf = make([]byte, 64)
	var lastTicks = make(map[string]LightStreamerTick, len(epics)) // epic -> tick

	defer close(tickReceiver)

	// map table index -> epic name
	var epicIndex = make(map[string]string, len(epics))
	for i, epic := range epics {
		epicIndex[fmt.Sprintf("1,%d", i+1)] = epic
	}

	for {
		read, err := resp.Body.Read(respBuf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("reading lightstreamer subscription failed: %v", err)
			break
		}

		priceMsg := string(respBuf[0:read])
		priceParts := strings.Split(priceMsg, "|")

		// Sever ends streaming
		if priceMsg == "LOOP\r\n\r\n" {
			fmt.Printf("ending\n")
			break
		}

		if len(priceParts) != len(fields)+1 {
			//fmt.Printf("Malformed price message: %q\n", priceMsg)
			continue
		}

		var parsedDate time.Time
		if priceParts[1] != "" {
			priceTime := priceParts[1]
			now := time.Now().UTC()
			i, err := strconv.ParseInt(priceTime, 10, 64)
			if err != nil {
				panic(err)
			}
			parsedTime := time.Unix(0, time.Now().Unix()+i*int64(time.Millisecond))

			parsedDate, err = time.ParseInLocation("2006-1-2 15:04:05", fmt.Sprintf("%d-%d-%d %s",
				now.Year(), now.Month(), now.Day(), parsedTime.Format("15:04:05")), time.Local)
			if err != nil {
				fmt.Printf("parsing time failed: %v time=%q\n", err, priceTime)
				continue
			}
		}
		tableIndex := priceParts[0]
		priceOpen, _ := strconv.ParseFloat(priceParts[2], 64)
		priceHigh, _ := strconv.ParseFloat(priceParts[3], 64)
		priceLow, _ := strconv.ParseFloat(priceParts[4], 64)
		priceClose, err := strconv.ParseFloat(strings.Trim(priceParts[5], "\r\n"), 64)

		epic, found := epicIndex[tableIndex]
		if !found {
			epic = epicNameUnknown
		}

		if epic != epicNameUnknown {
			var lastTick, found = lastTicks[epic]
			if found {
				if priceHigh == 0 {
					priceHigh = lastTick.MaxPrice
				}
				if priceOpen == 0 {
					priceOpen = lastTick.OpenPrice
				}
				if priceLow == 0 {
					priceLow = lastTick.MinPrice
				}
				if priceClose == 0 {
					priceClose = lastTick.ClosePrice
				}
				if parsedDate.IsZero() {
					parsedDate = lastTick.Time
				}
			}
		}

		tick := LightStreamerTick{
			Epic:       epic,
			Time:       parsedDate,
			OpenPrice:  priceOpen,
			MaxPrice:   priceHigh,
			MinPrice:   priceLow,
			ClosePrice: priceClose,
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
