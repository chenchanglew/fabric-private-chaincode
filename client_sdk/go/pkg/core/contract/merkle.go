package contract

import (
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func getPeerAddrs() ([]string, error) {
	// TODO get peer Address without hardcoded.
	peerAddrs := []string{}
	// peer1, peer2
	peerAddrs = append(peerAddrs, "http://127.0.0.1:21004")
	peerAddrs = append(peerAddrs, "http://127.0.0.1:21011")
	return peerAddrs, nil
}

func getMerkleRootFromPeer(peerAddr string, namespace string) ([]byte, error) {
	// TODO use secure communication to get merkleroot
	request := peerAddr + "/merkleRoot?namespace=" + namespace
	response, err := http.Get(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		log.Fatalf("Request failed with status code %d", response.StatusCode)
	}
	return body, nil
}

func GetMerkleRoots(namespace string) (string, error) {
	peerAddrs, err := getPeerAddrs()
	if err != nil {
		return "", err
	}
	merkleRoots := make([]string, len(peerAddrs))

	for i, p := range peerAddrs {
		m, err := getMerkleRootFromPeer(p, namespace)
		if err != nil {
			return "", err
		}
		merkleRoots[i] = hex.EncodeToString(m)
	}
	return strings.Join(merkleRoots, "|"), nil
}

func AddMerkleRootToArgs(args *[]string, namespace string) error {
	useMerkle, _ := strconv.ParseBool(os.Getenv("FPC_MERKLE_SOLUTION"))
	if !useMerkle {
		return nil
	}

	merkleRootStrs, err := GetMerkleRoots(namespace)
	if err != nil {
		return err
	}
	if len(merkleRootStrs) > 0 {
		*args = append(*args, merkleRootStrs)
	}
	return nil
}
