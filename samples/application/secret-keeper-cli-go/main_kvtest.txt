/*
Copyright IBM Corp. All Rights Reserved.
Copyright 2020 Intel Corporation

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-private-chaincode/samples/application/simple-cli-go/pkg"
)

func initConfig() *pkg.Config {

	getStrEnv := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			panic(fmt.Sprintf("%s not set", key))
		}
		return val
	}

	getBoolEnv := func(key string) bool {
		val := getStrEnv(key)
		ret, err := strconv.ParseBool(val)
		if err != nil {
			if val == "" {
				panic(fmt.Sprintf("invalid bool value for %s", key))
			}
		}
		return ret
	}

	config := &pkg.Config{
		CorePeerAddress:         getStrEnv("CORE_PEER_ADDRESS"),
		CorePeerId:              getStrEnv("CORE_PEER_ID"),
		CorePeerLocalMSPID:      getStrEnv("CORE_PEER_LOCALMSPID"),
		CorePeerMSPConfigPath:   getStrEnv("CORE_PEER_MSPCONFIGPATH"),
		CorePeerTLSCertFile:     getStrEnv("CORE_PEER_TLS_CERT_FILE"),
		CorePeerTLSEnabled:      getBoolEnv("CORE_PEER_TLS_ENABLED"),
		CorePeerTLSKeyFile:      getStrEnv("CORE_PEER_TLS_KEY_FILE"),
		CorePeerTLSRootCertFile: getStrEnv("CORE_PEER_TLS_ROOTCERT_FILE"),
		OrdererCA:               getStrEnv("ORDERER_CA"),
		ChaincodeId:             getStrEnv("CC_NAME"),
		ChannelId:               getStrEnv("CHANNEL_NAME"),
		GatewayConfigPath:       getStrEnv("GATEWAY_CONFIG"),
	}
	return config
}

func clientSend(client *pkg.Client, sendNum int, keyPrefix string, finish chan string, data chan []string, latency chan time.Duration) {
	numOpRep := 2
	numKey := sendNum / numOpRep

	latencyArr := make([]string, sendNum)
	totalDuration := 0 * time.Second
	for i := 0; i < sendNum; i++ {
		keyIndex := i
		if keyIndex >= numKey {
			keyIndex -= numKey
		}
		key := keyPrefix + "_" + strconv.Itoa(keyIndex) + "RequiredLongPostfixHereToMatchRequirement"
		start := time.Now()
		if (i/numKey)%(2) == 0 {
			// put state: value = key
			_ = client.Invoke("put_state", key, key+"_"+strconv.Itoa(i))
			// fmt.Println("> " + res)
		} else {
			// get state: expect value = key
			_ = client.Invoke("get_state", key)
			// fmt.Println("> " + res)
		}
		runDuration := time.Since(start)
		latencyArr[i] = fmt.Sprintf("%d", runDuration.Milliseconds())
		totalDuration += runDuration
	}
	avgLatency := totalDuration / time.Duration(sendNum)

	finish <- keyPrefix
	latency <- avgLatency
	data <- latencyArr
}

func runExperiment(config *pkg.Config, clientNum int, numReqEach int, filename string) {
	clientSet := make([]*pkg.Client, clientNum)

	for i := 0; i < clientNum; i++ {
		clientSet[i] = pkg.NewClient(config)
	}
	fmt.Println("clientset:", clientSet)

	finishChan := make(chan string, 1)
	dataChan := make(chan []string, 1)
	latencyChan := make(chan time.Duration, 1)

	start := time.Now()
	for i := 0; i < clientNum; i++ {
		keyPrefix := "c" + strconv.Itoa(i)
		go clientSend(clientSet[i], numReqEach, keyPrefix, finishChan, dataChan, latencyChan)
	}

	collect := 0
	for collect < clientNum {
		select {
		case <-finishChan:
			collect += 1
		case <-time.After(30 * time.Second):
			fmt.Println("Still waiting for completion, has run:", time.Since(start))
		}
	}
	totalDuration := time.Since(start)
	fmt.Println("Finish", clientNum*numReqEach, "of requests, duration:", totalDuration)

	collect = 0
	totalLatency := 0 * time.Millisecond
	for collect < clientNum {
		latency := <-latencyChan
		totalLatency += latency
		collect += 1
	}
	fmt.Printf("Throughput(Txn/s): %4f, Avglatency(ms/req): %v\n", float32(clientNum*numReqEach*1000)/float32(totalDuration.Milliseconds()), totalLatency/time.Duration(clientNum))

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	collect = 0
	for collect < clientNum {
		clientData := <-dataChan
		writer.Write(clientData)
		collect += 1
	}

}

func main() {
	config := initConfig()
	solution := "SKVS"
	repeatTime := 3
	clientNum := 2
	numReqEach := 700
	for i := 0; i < repeatTime; i++ {
		filename := fmt.Sprintf("%s_latency_%d_%d_%d.csv", solution, clientNum, numReqEach, i)
		filepath := filepath.Join("/Users/lew/Desktop/fpc-notes/misc/latencyExp/", filename)
		runExperiment(config, clientNum, numReqEach, filepath)
		time.Sleep(3 * time.Second)
		runtime.GC()
		time.Sleep(3 * time.Second)
	}
}
