package main

import (
	"fmt"
	"time"

	"github.com/amaurybrisou/igmarkets"
	"github.com/lfritz/env"
	log "github.com/sirupsen/logrus"
)

var conf struct {
	igAPIURL     string
	igIdentifier string
	igAPIKey     string
	igPassword   string
	igAccountID  string
	instrument   string
}

func main() {
	var e = env.New()
	e.OptionalString("INSTRUMENT", &conf.instrument, "CS.D.EURUSD.MINI.IP", "instrument to trade")
	e.OptionalString("IG_API_URL", &conf.igAPIURL, igmarkets.DemoAPIURL, "IG API URL")
	e.OptionalString("IG_IDENTIFIER", &conf.igIdentifier, "", "IG Identifier")
	e.OptionalString("IG_API_KEY", &conf.igAPIKey, "", "IG API key")
	e.OptionalString("IG_PASSWORD", &conf.igPassword, "", "IG password")
	e.OptionalString("IG_ACCOUNT", &conf.igAccountID, "", "IG account ID")
	if err := e.Load(); err != nil {
		log.WithError(err).Fatal("env loading failed")
	}

	igHandle, _ := igmarkets.New(conf.igAPIURL, conf.igAPIKey, conf.igAccountID, conf.igIdentifier, conf.igPassword, time.Second*30)
	if err := igHandle.Login(); err != nil {
		log.WithError(err).Error("new fialed")
		return
	}

	tickChan := make(chan igmarkets.LightStreamChartTick)
	err := igHandle.OpenLightStreamerSubscription([]string{"CS.D.ETHUSD.CFD.IP"}, []string{
		"UTM",
		"OFR_OPEN",
		"OFR_HIGH",
		"OFR_LOW",
		"OFR_CLOSE",
	}, "CHART", "SECOND", "MERGE", tickChan)
	if err != nil {
		log.WithError(err).Error("open stream fialed")
	}

	run := func(stop chan bool) {
		for tick := range tickChan {
			log.Infof("tick: %+v", tick)
		}
		log.Info("run ended")
	}

	stop := make(chan bool)
	go run(stop)

	log.Infof("Server closed stream, restarting...")

	<-time.After(50 * time.Second)

	// log.Info("sending stop")
	// stop <- true
	// log.Info("unsubscribe")
	err = retry(5, time.Second*5, func() error { return igHandle.CloseLightStreamerSubscription() })
	if err != nil {
		log.Error(err)
	}

	for tick := range tickChan {
		log.Infof("tick !!!: %+v", tick)
	}
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		log.Println("retrying after error:", err)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
