package contract

import (
	"context"
	"encoding/hex"
	"os"
	"strings"

	mtcs "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/mtreecomp/mtreegrpc"
	"google.golang.org/grpc"
)

func getPeerAddrs() ([]string, error) {
	// TODO get peer Address without hardcoded.
	peerAddrs := []string{}
	// peer1, peer2
	peerAddrs = append(peerAddrs, "127.0.0.1:28884")
	peerAddrs = append(peerAddrs, "127.0.0.1:28885")
	return peerAddrs, nil
}

func getMerkleRootFromPeer(peerAddr string, namespace string) ([]byte, error) {
	// TODO use secure communication to get merkleroot
	conn, err := grpc.Dial(peerAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := mtcs.NewMerkleServiceClient(conn)
	response, err := client.GetMerkleRoot(context.Background(), &mtcs.MerkleRootRequest{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	return proto.Marshal(response)
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
	useMerkle := os.Getenv("FPC_Merkle_Solution")
	if useMerkle != "True" {
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
