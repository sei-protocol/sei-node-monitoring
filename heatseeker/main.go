package main

import (
	"flag"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.InfoLevel)
	var vortexEndpoint string
	flag.StringVar(&vortexEndpoint, "vortex", "", "vortex endpoint to check")
	flag.Parse()

	client := &http.Client{}
	for {
		resp, err := client.Get(vortexEndpoint)
		if err != nil {
			log.WithFields(log.Fields{
				"vortex": vortexEndpoint,
				"error":  err}).Warning("Unable to query endpoint")
		}
		defer resp.Body.Close()
		if !(resp.StatusCode == 200) {
			log.WithFields(log.Fields{
				"status code": resp.StatusCode}).Warning("Didn't receive 200 status code")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"body":  body,
				"error": err}).Warning("Unable to parse body")
		}
		time.Sleep(5 * time.Second)
	}
}
