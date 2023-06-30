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
	// TODO how to get peer address
	panic("Not implemented")
}

func getMerkleRootFromPeer(peerAddr string) ([]byte, error) {
	// TODO use secure communication to get merkleroot
	conn, err := grpc.Dial(peerAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// TODO get namespace without hardcoded
	namespace := "fpc-secret-keeper-go"

	client := mtcs.NewMerkleServiceClient(conn)
	response, err := client.GetMerkleRoot(context.Background(), &mtcs.MerkleRootRequest{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	return response.GetData(), nil
}

func GetMerkleRoots() (string, error) {
	peerAddrs, err := getPeerAddrs()
	if err != nil {
		return "", err
	}
	merkleRoots := make([]string, len(peerAddrs))

	for i, p := range peerAddrs {
		m, err := getMerkleRootFromPeer(p)
		if err != nil {
			return "", err
		}
		merkleRoots[i] = hex.EncodeToString(m)
	}
	return strings.Join(merkleRoots, "|"), nil
}

func AddMerkleRootToArgs(args *[]string) error {
	useMerkle := os.Getenv("FPC_Merkle_Solution")
	if useMerkle != "True" {
		return nil
	}

	merkleRootStrs, err := GetMerkleRoots()
	if err != nil {
		return err
	}
	if len(merkleRootStrs) > 0 {
		*args = append(*args, merkleRootStrs)
	}
	return nil
}
