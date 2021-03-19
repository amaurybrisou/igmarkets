package igmarkets

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	contentLength = "100000000"
	contentType   = "application/x-www-form-urlencoded"
)

type LightStreamerTick struct {
	Epic             string
	Time             time.Time
	OpenPrice        float64
	MaxPrice         float64
	MinPrice         float64
	ClosePrice       float64
	LastTradedVolume float64
}

type LightStreamOptions struct {
	Epics, Fields                     []string
	SubType, Interval, Mode           string
	ReconnectionTime, MaxReconnection int
}

func (ig *IGMarkets) LogoutLightStreamer() error {
	err := ig.CloseLightStreamerSubscription()
	if err != nil {
		return err
	}

	err = ig.Logout()
	if err != nil {
		return err
	}

	return nil
}

func (ig *IGMarkets) CloseLightStreamerSubscription() error {

	const contentType = "application/x-www-form-urlencoded"

	tr := &http.Transport{
		MaxIdleConns:          1,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    true,
	}
	c := &http.Client{Transport: tr, Timeout: 30 * time.Second}

	body := []byte(fmt.Sprintf("LS_session=%s&LS_op=destroy", ig.SessionID))
	url := fmt.Sprintf("%s/lightstreamer/control.txt", ig.SessionVersion2.LightstreamerEndpoint)
	resp, err := c.Post(url, contentType, bytes.NewBuffer(body))
	if err != nil {
		return LightStreamErrorHandler(resp, err)
	}
	defer resp.Body.Close()

	bodyResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return LightStreamErrorHandler(resp, err)
	}

	sessionMsg := strings.Trim(string(bodyResp[:]), "\n")

	if !strings.HasPrefix(string(sessionMsg), "OK") {
		return fmt.Errorf("unexpected response from lightstreamer session endpoint %q: %q", url, string(sessionMsg))
	}

	log.Debug("lightstreamer: subscription closed")

	return nil

}

func (ig *IGMarkets) connectLightStreamer(options LightStreamOptions) (*http.Response, error) {
	if err := ig.Login(); err != nil {
		return nil, err
	}

	// Obtain CST and XST tokens first
	sessionVersion2, err := ig.LoginVersion2()
	if err != nil {
		return nil, fmt.Errorf("ig.LoginVersion2() failed: %v", err)
	}

	log.Debug("lightstreamer : connected")

	ig.SessionVersion2 = *sessionVersion2

	tr := &http.Transport{
		MaxIdleConns:       5,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	c := &http.Client{Transport: tr}

	bodyAsStr := "LS_polling=true&LS_polling_millis=0&LS_idle_millis=0" +
		"&LS_op2=create&LS_password=CST-" +
		sessionVersion2.CSTToken + "|" + "XST-" + sessionVersion2.XSTToken + "&LS_user=" +
		sessionVersion2.CurrentAccountId + "&LS_cid=mgQkwtwdysogQz2BJ4Ji kOj2Bg"

	// Create Lightstreamer Session
	body := []byte(bodyAsStr)

	bodyBuf := bytes.NewBuffer(body)
	url := fmt.Sprintf("%s/lightstreamer/create_session.txt", sessionVersion2.LightstreamerEndpoint)
	resp, err := c.Post(url, contentType, bodyBuf)
	if err != nil {
		return nil, LightStreamErrorHandler(resp, err)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)

	sessionMsg := string(respBody[:])

	if !strings.HasPrefix(sessionMsg, "OK") {
		return nil, fmt.Errorf("unexpected response from lightstreamer session endpoint %q: %q", url, sessionMsg)
	}
	sessionParts := strings.Split(sessionMsg, "\r\n")
	sessionID := sessionParts[1]
	sessionID = strings.ReplaceAll(sessionID, "SessionId:", "")
	ig.SessionID = sessionID

	if len(sessionParts) > 2 {
		controlAddr := sessionParts[2]
		controlAddr = strings.ReplaceAll(controlAddr, "ControlAddress:", "")
		if controlAddr != "" {
			ig.SessionVersion2.LightstreamerEndpoint = "https://" + controlAddr
		}
	}

	log.Debug("lightstreamer : session created")

	// Adding subscription for epic
	var epicList string
	for i := range options.Epics {
		epicList = epicList + options.SubType + ":" + options.Epics[i] + ":" + options.Interval + "+"
	}

	body = []byte("LS_session=" + sessionID +
		"&LS_polling=true&LS_polling_millis=0&LS_idle_millis=0&LS_op=add&LS_Table=1&LS_id=" +
		epicList + "&LS_schema=" + strings.Join(options.Fields[:], "+") + "&LS_mode=" + options.Mode)
	bodyBuf = bytes.NewBuffer(body)
	url = fmt.Sprintf("%s/lightstreamer/control.txt", sessionVersion2.LightstreamerEndpoint)
	resp, err = c.Post(url, contentType, bodyBuf)

	if err != nil {
		return nil, LightStreamErrorHandler(resp, err)
	}

	body, _ = ioutil.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "OK") {
		return nil, fmt.Errorf("unexpected control.txt response: %q", body)
	}

	log.Debug("lightstreamer : subscription created")

	// Binding to subscription
	body = []byte("LS_session=" + sessionID + "&LS_polling=false&LS_content_length=" + contentLength)
	bodyBuf = bytes.NewBuffer(body)
	url = fmt.Sprintf("%s/lightstreamer/bind_session.txt", sessionVersion2.LightstreamerEndpoint)
	resp, err = c.Post(url, contentType, bodyBuf)
	if err != nil {
		return nil, LightStreamErrorHandler(resp, err)
	}

	log.Debug("lightstreamer : subscription bound")

	return resp, nil
}

// GetOTCWorkingOrders - Get all working orders
// epic: e.g. CS.D.BITCOIN.CFD.IP
// tickReceiver: receives all ticks from lightstreamer API
func (ig *IGMarkets) OpenLightStreamerSubscription(
	ctx context.Context,
	o LightStreamOptions) (<-chan LightStreamChartTick, <-chan error, error) {

	tickChan := make(chan LightStreamChartTick)
	errChan := make(chan error)

	go func() {
		attempts := 1

		defer close(tickChan)
		defer close(errChan)

		for attempts < o.MaxReconnection {

			resp, err := ig.connectLightStreamer(o)

			if err != nil {
				errChan <- err
				attempts++
				time.Sleep(time.Duration(attempts) * time.Duration(o.ReconnectionTime) * time.Second)
				continue
			}

			internalErrChan := make(chan error)
			internalTickChan := make(chan LightStreamChartTick)

			go readLightStreamSubscription(o.Epics, o.Fields, internalTickChan, resp.Body, internalErrChan)

			var wg sync.WaitGroup
			stop := make(chan bool)

			wg.Add(1)

			go func() {
				defer wg.Done()

				for {
					select {
					case t := <-internalTickChan:
						if t.UTM != nil {
							tickChan <- t
						}
					case <-stop:
						log.Debug("lightstreamer: stopping stream")
						return
					case <-ctx.Done():
						log.Debug("lightstreamer: stopping stream")
						return
					}
				}
			}()

			select {
			case err := <-internalErrChan:
				log.WithError(err).Error("lightstreamer : ")

				stop <- true

				err = ig.LogoutLightStreamer()
				if err != nil {
					log.WithError(err).Error("lightstreamer : ")
				}

				wg.Wait()

				log.Printf("sleeping for %s", time.Second*time.Duration(o.ReconnectionTime*attempts))
				time.Sleep(time.Second * time.Duration(o.ReconnectionTime*attempts))

			case <-ctx.Done():
				err := ig.LogoutLightStreamer()
				if err != nil {
					log.WithError(err).Error("lightstreamer : ")
				}

				wg.Wait()
				log.Debug("lightstreamer: stopping stream restarter")
				return
			}
		}
		log.Error("lightsreamer : stopping...")
		err := ig.LogoutLightStreamer()
		if err != nil {
			log.WithError(err).Error("lightstreamer : ")
		}
	}()

	return tickChan, errChan, nil
}
