package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/hyperledger/fabric-private-chaincode/tle_go/tlegrpc"
	"google.golang.org/grpc"
)

type TleServer struct {
	pb.UnimplementedTleServiceServer
	tleState *Tlestate
}

func (s *TleServer) GetMeta(ctx context.Context, req *pb.MetaRequest) (*pb.MetaResponse, error) {
	namespace := req.Namespace
	key := req.Key
	fmt.Printf("--- tle_go/server.go getMeta, namespace = %s, key = %s ---\n", namespace, key)

	data, err := s.tleState.GetMeta(namespace, key)
	if err != nil {
		return nil, err
	}

	lastCommitHash := []byte("Sample Commit Hash")
	return &pb.MetaResponse{Data: data, LastCommitHash: lastCommitHash}, nil
}

func (s *TleServer) GetSession(ctx context.Context, req *pb.Empty) (*pb.MetaResponse, error) {
	fmt.Printf("--- tle_go/server.go getSession ---\n")
	data := []byte("")
	lastCommitHash := []byte("Sample Commit Hash")
	return &pb.MetaResponse{Data: data, LastCommitHash: lastCommitHash}, nil
}

func ServeMeta(port string, tleState *Tlestate) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTleServiceServer(s, &TleServer{tleState: tleState})
	fmt.Println("Server listening on port", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
