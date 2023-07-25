package listener

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/hyperledger/fabric-config/protolator"
	cb "github.com/hyperledger/fabric-protos-go/common"
	ab "github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/protoutil"
	"google.golang.org/grpc"
)

var (
	oldest  = &ab.SeekPosition{Type: &ab.SeekPosition_Oldest{Oldest: &ab.SeekOldest{}}}
	newest  = &ab.SeekPosition{Type: &ab.SeekPosition_Newest{Newest: &ab.SeekNewest{}}}
	maxStop = &ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: math.MaxUint64}}}
)

type deliverClient struct {
	client    ab.AtomicBroadcast_DeliverClient
	channelID string
	signer    SignerSerializer
	quiet     bool
}

func newDeliverClient(client ab.AtomicBroadcast_DeliverClient, channelID string, signer SignerSerializer, quiet bool) *deliverClient {
	return &deliverClient{client: client, channelID: channelID, signer: signer, quiet: quiet}
}

func (r *deliverClient) seekHelper(start *ab.SeekPosition, stop *ab.SeekPosition) *cb.Envelope {
	env, err := protoutil.CreateSignedEnvelope(cb.HeaderType_DELIVER_SEEK_INFO, r.channelID, r.signer, &ab.SeekInfo{
		Start:    start,
		Stop:     stop,
		Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
	}, 0, 0)
	if err != nil {
		panic(err)
	}
	return env
}

func (r *deliverClient) seekOldest() error {
	return r.client.Send(r.seekHelper(oldest, maxStop))
}

func (r *deliverClient) seekNewest() error {
	return r.client.Send(r.seekHelper(newest, maxStop))
}

func (r *deliverClient) seekSingle(blockNumber uint64) error {
	specific := &ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: blockNumber}}}
	return r.client.Send(r.seekHelper(specific, specific))
}

func (r *deliverClient) readUntilClose(blockChan chan cb.Block) {
	for {
		msg, err := r.client.Recv()
		if err != nil {
			fmt.Println("Error receiving:", err)
			return
		}

		switch t := msg.Type.(type) {
		case *ab.DeliverResponse_Status:
			fmt.Println("Got status ", t)
			return
		case *ab.DeliverResponse_Block:
			fmt.Println("Received block: ", t.Block.Header.Number)
			blockChan <- *t.Block

			if !r.quiet {
				fmt.Scanln() // wait for Enter Key
				err := protolator.DeepMarshalJSON(os.Stdout, t.Block)
				if err != nil {
					fmt.Printf("  Error pretty printing block: %s", err)
				}
			}
		}
	}
}

func ListenBlock(channelID string, serverAddr string, seek int, quiet bool, caCertPath string, blockChan chan cb.Block) {

	// IF meet config load problem make sure two things:
	// 1. orderer.yaml able to find.
	// 2. ENV FABRIC_CFG_PATH is empty, or else can run with FABRIC_CFG_PATH=. go run .
	conf, err := localconfig.Load()
	if err != nil {
		fmt.Println("failed to load config:", err)
		os.Exit(1)
	}
	// fmt.Println("conf:", conf)
	fmt.Println("localMSPDir:", conf.General.LocalMSPDir)
	fmt.Println("BCCSP:", conf.General.BCCSP)
	fmt.Println("localMSPID:", conf.General.LocalMSPID)

	// Load local MSP
	mspConfig, err := msp.GetLocalMspConfig(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
	if err != nil {
		fmt.Println("Failed to load MSP config:", err)
		os.Exit(0)
	}
	fmt.Println("mspConfig:", mspConfig)
	fmt.Println("factoryGetDefault:", factory.GetDefault())
	err = mspmgmt.GetLocalMSP(factory.GetDefault()).Setup(mspConfig)
	if err != nil { // Handle errors reading the config file
		fmt.Println("Failed to initialize local MSP:", err)
		os.Exit(0)
	}

	signer, err := mspmgmt.GetLocalMSP(factory.GetDefault()).GetDefaultSigningIdentity()
	if err != nil {
		fmt.Println("Failed to load local signing identity:", err)
		os.Exit(0)
	}

	if seek < -2 {
		fmt.Println("Wrong seek value.")
		os.Exit(0)
	}

	tlsCredentials, err := loadTLSCredentials(caCertPath)
	if err != nil {
		fmt.Println("Error loading TLS Credentials:", err)
		return
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(tlsCredentials))
	// conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	client, err := ab.NewAtomicBroadcastClient(conn).Deliver(context.TODO())
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	s := newDeliverClient(client, channelID, signer, quiet)
	switch seek {
	case -2:
		err = s.seekOldest()
	case -1:
		err = s.seekNewest()
	default:
		err = s.seekSingle(uint64(seek))
	}

	if err != nil {
		fmt.Println("Received error:", err)
	}

	s.readUntilClose(blockChan)
}

func main() {
	channelID := "testchannel"
	serverAddr := "127.0.0.1:20000"
	seek := -2     // -2 is load from oldest, -1 load from newest, other int -> load from the int.
	quiet := false // ture = only print block number, false = print whole block.
	caCertPath := "/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/crypto/ca-certs.pem"
	blockChan := make(chan cb.Block, 10)

	ListenBlock(channelID, serverAddr, seek, quiet, caCertPath, blockChan)
}

// func main() {
// 	conf, err := localconfig.Load()
// 	if err != nil {
// 		fmt.Println("failed to load config:", err)
// 		os.Exit(1)
// 	}

// 	// Load local MSP
// 	mspConfig, err := msp.GetLocalMspConfig(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
// 	if err != nil {
// 		fmt.Println("Failed to load MSP config:", err)
// 		os.Exit(0)
// 	}
// 	err = mspmgmt.GetLocalMSP(factory.GetDefault()).Setup(mspConfig)
// 	if err != nil { // Handle errors reading the config file
// 		fmt.Println("Failed to initialize local MSP:", err)
// 		os.Exit(0)
// 	}

// 	signer, err := mspmgmt.GetLocalMSP(factory.GetDefault()).GetDefaultSigningIdentity()
// 	if err != nil {
// 		fmt.Println("Failed to load local signing identity:", err)
// 		os.Exit(0)
// 	}

// 	var channelID string
// 	var serverAddr string
// 	var seek int
// 	var quiet bool

// 	flag.StringVar(&serverAddr, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
// 	flag.StringVar(&channelID, "channelID", "mychannel", "The channel ID to deliver from.")
// 	flag.BoolVar(&quiet, "quiet", false, "Only print the block number, will not attempt to print its block contents.")
// 	flag.IntVar(&seek, "seek", -2, "Specify the range of requested blocks."+
// 		"Acceptable values:"+
// 		"-2 (or -1) to start from oldest (or newest) and keep at it indefinitely."+
// 		"N >= 0 to fetch block N only.")
// 	flag.Parse()

// 	if seek < -2 {
// 		fmt.Println("Wrong seek value.")
// 		flag.PrintDefaults()
// 	}
// 	serverAddr = "127.0.0.1:20000"
// 	channelID = "testchannel"
// 	fmt.Println("serverAddr:", serverAddr)
// 	fmt.Println("channelID:", channelID)

// 	tlsCredentials, err := loadTLSCredentials()

// 	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(tlsCredentials))
// 	// conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	// conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
// 	if err != nil {
// 		fmt.Println("Error connecting:", err)
// 		return
// 	}
// 	client, err := ab.NewAtomicBroadcastClient(conn).Deliver(context.TODO())
// 	if err != nil {
// 		fmt.Println("Error connecting:", err)
// 		return
// 	}

// 	s := newDeliverClient(client, channelID, signer, quiet)
// 	switch seek {
// 	case -2:
// 		err = s.seekOldest()
// 	case -1:
// 		err = s.seekNewest()
// 	default:
// 		err = s.seekSingle(uint64(seek))
// 	}

// 	if err != nil {
// 		fmt.Println("Received error:", err)
// 	}

// 	s.readUntilClose()
// }
