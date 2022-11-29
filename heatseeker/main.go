package main

import (
	"context"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"strings"
	"time"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var FAUCET_ADDRESSES = []string{"sei1dhwul4rz8jfwvenqpyhdctax2tuljk2ag0v864"}

func runVortexFECheck(client *http.Client, vortexEndpoint string) {
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
		ReportVortexFEMetrics(resp.StatusCode)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err}).Warning("Unable to parse body")
	}
	ReportVortexFEMetrics(resp.StatusCode)

}

func runOffchainIndexerCheck(client *http.Client, endpoints []string) {
	for _, endpoint := range endpoints {
		resp, err := client.Get(endpoint)
		if err != nil {
			log.WithFields(log.Fields{
				"indexer_endpoint": endpoint,
				"error":  err}).Warning("Unable to query endpoint")
		}
		if !(resp.StatusCode == 200) {
			log.WithFields(log.Fields{
				"status code": resp.StatusCode}).Warning("Didn't receive 200 status code")
			ReportIndexerMetrics(endpoint, resp.StatusCode)
			resp.Body.Close()
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"body":  body,
				"error": err}).Warning("Unable to parse body")
		}
		ReportIndexerMetrics(endpoint, resp.StatusCode)
		resp.Body.Close()
	}

}

func runFaucetCheck(grpcConn *grpc.ClientConn, faucetAddrs []string) {
	bankClient := banktypes.NewQueryClient(grpcConn)
	for _, addr := range faucetAddrs {
		bankRes, err := bankClient.AllBalances(
			context.Background(),
			&banktypes.QueryAllBalancesRequest{Address: addr},
		)
		if err != nil {
			log.Error(fmt.Sprintf("Could not get balance for faucet %s", addr), err)
			continue
		}
		for _, balance := range bankRes.Balances {
			ReportFaucetMetrics(addr, float32(balance.Amount.Int64()), balance.Denom)
		}
	}
}

func main() {
	log.SetLevel(log.InfoLevel)
	var vortexEndpoint string
	var nodeAddress string
	var faucetAddrs string
	var offchainIndexerEndpoints string
	flag.StringVar(&vortexEndpoint, "vortex", "", "vortex endpoint to check")
	flag.StringVar(&nodeAddress, "node-address", "", "node address to check")
	flag.StringVar(&faucetAddrs, "faucet-addrs", "", "comma separated list of faucet addrs to check")
	flag.StringVar(&offchainIndexerEndpoints, "offchain-indexer", "", "comma separated list of offchain indexer endpoints to check")
	flag.Parse()

	// Start metrics collector in another thread
	metricsServer := MetricsServer{}
	go metricsServer.StartMetricsClient()

	client := &http.Client{}
	grpcConn, err := grpc.Dial(
		nodeAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatal("Could not connect to gRPC node")
	}

	for {
		log.Info("Running")
		time.Sleep(60 * time.Second)
		// Vortex Frontend Check
		runVortexFECheck(client, vortexEndpoint)
		// Sei Faucet Check
		runFaucetCheck(grpcConn, strings.Split(faucetAddrs, ","))
		// Offchain Indexer Serverless Query Endpoint Check
		runOffchainIndexerCheck(client, strings.Split(offchainIndexerEndpoints, ","))
	}
}
