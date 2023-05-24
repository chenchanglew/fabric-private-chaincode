package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-private-chaincode/tle_go/mocks"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	"github.com/hyperledger/fabric/core/handlers/library"
	validation "github.com/hyperledger/fabric/core/handlers/validation/api"
	ledgermocks "github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func SetupConfig(t *testing.T) {
	// cwd, err := os.Getwd()
	cwd := "/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/peers/"
	// require.NoError(t, err, "failed to get current working directory")

	viper.SetConfigFile(filepath.Join(cwd, "core.yaml"))
	viper.ReadInConfig()

	//Capture the configuration from viper
	viper.Set("peer.addressAutoDetect", false)
	viper.Set("peer.address", "localhost:8080")
	viper.Set("peer.id", "testPeerID")
	viper.Set("peer.localMspId", "Org1MSP")
	viper.Set("peer.listenAddress", "0.0.0.0:7051")
	viper.Set("peer.authentication.timewindow", "15m")
	viper.Set("peer.tls.enabled", "false")
	viper.Set("peer.networkId", "testNetwork")
	viper.Set("peer.limits.concurrency.endorserService", 2500)
	viper.Set("peer.limits.concurrency.deliverService", 2500)
	viper.Set("peer.discovery.enabled", true)
	viper.Set("peer.profile.enabled", false)
	viper.Set("peer.profile.listenAddress", "peer.authentication.timewindow")
	viper.Set("peer.discovery.orgMembersAllowedAccess", false)
	viper.Set("peer.discovery.authCacheEnabled", true)
	viper.Set("peer.discovery.authCacheMaxSize", 1000)
	viper.Set("peer.discovery.authCachePurgeRetentionRatio", 0.75)
	viper.Set("peer.chaincodeListenAddress", "0.0.0.0:7052")
	viper.Set("peer.chaincodeAddress", "0.0.0.0:7052")
	viper.Set("peer.validatorPoolSize", 1)

	viper.Set("vm.endpoint", "unix:///var/run/docker.sock")
	viper.Set("vm.docker.tls.enabled", false)
	viper.Set("vm.docker.attachStdout", false)
	viper.Set("vm.docker.hostConfig.NetworkMode", "TestingHost")
	viper.Set("vm.docker.tls.cert.file", "test/vm/tls/cert/file")
	viper.Set("vm.docker.tls.key.file", "test/vm/tls/key/file")
	viper.Set("vm.docker.tls.ca.file", "test/vm/tls/ca/file")

	viper.Set("operations.listenAddress", "127.0.0.1:9443")
	viper.Set("operations.tls.enabled", false)
	viper.Set("operations.tls.cert.file", "test/tls/cert/file")
	viper.Set("operations.tls.key.file", "test/tls/key/file")
	viper.Set("operations.tls.clientAuthRequired", false)
	viper.Set("operations.tls.clientRootCAs.files", []string{"relative/file1", "/absolute/file2"})

	viper.Set("metrics.provider", "disabled")
	viper.Set("metrics.statsd.network", "udp")
	viper.Set("metrics.statsd.address", "127.0.0.1:8125")
	viper.Set("metrics.statsd.writeInterval", "10s")
	viper.Set("metrics.statsd.prefix", "testPrefix")

	viper.Set("chaincode.pull", false)
	viper.Set("chaincode.externalBuilders", &[]peer.ExternalBuilder{
		{
			Path: "relative/plugin_dir",
			Name: "relative",
		},
		{
			Path: "/absolute/plugin_dir",
			Name: "absolute",
		},
	})
}

func PrintConfig() {
	fmt.Println("--- viper config ---")
	settings := viper.AllSettings()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}
	fmt.Println("--finish viper config--")
}

func InitializePeer(peerInstance *peer.Peer) error {
	libConf, err := library.LoadConfig()
	if err != nil {
		return fmt.Errorf("could not decode peer handlers configuration [%s]", err)
	}

	reg := library.InitRegistry(libConf)
	validationPluginsByName := reg.Lookup(library.Validation).(map[string]validation.PluginFactory)
	peerInstance.Initialize(
		func(cid string) {
			fmt.Printf("--- peerInstance Init function with cid: %s ---\n", cid)
		},
		nil,
		plugin.MapBasedMapper(validationPluginsByName),
		&ledgermocks.DeployedChaincodeInfoProvider{},
		nil,
		nil,
		runtime.NumCPU(),
	)
	return nil
}

func SetupConfig2() {
	viper.SetConfigFile("/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/peers/Org1.Org1_peer_0/core.yaml")
	viper.ReadInConfig()
	viper.SetEnvPrefix("core")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	PrintConfig()
}

func TestBunchValidation(t *testing.T) {
	// setup config
	SetupConfig2()
	defer viper.Reset()

	// read genesis block
	rawBlock0, err := ioutil.ReadFile("blocks/t0.block")
	require.NoError(t, err)
	block0, err := protoutil.UnmarshalBlock(rawBlock0)
	require.NoError(t, err)

	// initialize peer
	// peerInstance, cleanup := peer.NewTestPeerLight(t)
	peerInstance, cleanup := peer.NewTestPeer2(t)
	defer cleanup()

	err = InitializePeer(peerInstance)
	require.NoError(t, err)

	fmt.Println("---- Creating liftcycleValidation ----")
	// legacyLifecycleValidation := (plugindispatcher.LifecycleResources)(nil)
	// NewLiftcycleValidation := (plugindispatcher.CollectionAndLifecycleResources)(nil)
	legacyLifecycleValidation, NewLifecycleValidation := createLifecycleValidation(peerInstance)
	channelName := "testchannel"

	fmt.Println("---- Creating channel ----")
	err = peerInstance.CreateChannel(channelName, block0, &ledgermocks.DeployedChaincodeInfoProvider{}, legacyLifecycleValidation, NewLifecycleValidation)
	if err != nil {
		t.Fatalf("failed to create chain %s", err)
	}

	policyMgr := policies.PolicyManagerGetterFunc(peerInstance.GetPolicyManager)

	verifyNum := 12

	fmt.Println("---- Creating validator ----")
	validator, err := CreateTxValidatorViaPeer(peerInstance, channelName, legacyLifecycleValidation, NewLifecycleValidation)
	require.NoError(t, err)

	for i := 1; i <= verifyNum; i++ {
		fmt.Printf("---verifying block %d----\n", i)
		rawBlockX, err := ioutil.ReadFile("blocks/t" + strconv.Itoa(i) + ".block")
		require.NoError(t, err)
		blockX, err := protoutil.UnmarshalBlock(rawBlockX)
		require.NoError(t, err)

		err = VerifyBlock(policyMgr, []byte("testchannel"), uint64(i), blockX)
		if err != nil {
			fmt.Println(err)
		}
		require.NoError(t, err)

		fmt.Printf("--- Verify Block %d success, start verify txn ---\n", i)

		err = validator.Validate(blockX)
		require.NoError(t, err)
	}

}

func TestIntegration(t *testing.T) {

	rawBlock, err := ioutil.ReadFile("blocks/t2.block")
	require.NoError(t, err)

	block, err := protoutil.UnmarshalBlock(rawBlock)
	require.NoError(t, err)

	// fmt.Println(block)
	// id_bytes_Org1MSP := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNsVENDQWp5Z0F3SUJBZ0lRSktkeTVvT2Vzd0VPYmk5S2o1cmNHREFLQmdncWhrak9QUVFEQWpCek1Rc3cKQ1FZRFZRUUdFd0pWVXpFVE1CRUdBMVVFQ0JNS1EyRnNhV1p2Y201cFlURVdNQlFHQTFVRUJ4TU5VMkZ1SUVaeQpZVzVqYVhOamJ6RVpNQmNHQTFVRUNoTVFiM0puTVM1bGVHRnRjR3hsTG1OdmJURWNNQm9HQTFVRUF4TVRZMkV1CmIzSm5NUzVsZUdGdGNHeGxMbU52YlRBZUZ3MHlNekExTVRZeE1qTTJNREJhRncwek16QTFNVE14TWpNMk1EQmEKTUZzeEN6QUpCZ05WQkFZVEFsVlRNUk13RVFZRFZRUUlFd3BEWVd4cFptOXlibWxoTVJZd0ZBWURWUVFIRXcxVApZVzRnUm5KaGJtTnBjMk52TVI4d0hRWURWUVFEREJaQlpHMXBia0J2Y21jeExtVjRZVzF3YkdVdVkyOXRNRmt3CkV3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFRUo1cUkrS1pIazRnUmRCekd0TFdoZU5TSytCamZWRWQKQURxS0x4djZBVklySXZ6bEI0NTRtUVZRVjh3dStibkhUdnIvakJBQWpmUit2V25vQ2hmVzU2T0J5VENCeGpBTwpCZ05WSFE4QkFmOEVCQU1DQjRBd0RBWURWUjBUQVFIL0JBSXdBREFyQmdOVkhTTUVKREFpZ0NCN3FxSWtyOFdnCm10aDl2UnRwOWw5VFJKUGY4RnBGaUE2Q1owREczOTRSc1RCNUJnZ3FBd1FGQmdjSUFRUnRleUpoZEhSeWN5STYKZXlKb1ppNUJabVpwYkdsaGRHbHZiaUk2SWlJc0ltaG1Ma1Z1Y205c2JHMWxiblJKUkNJNklrRmtiV2x1UUc5eQpaekV1WlhoaGJYQnNaUzVqYjIwaUxDSm9aaTVVZVhCbElqb2lZMnhwWlc1MElpd2ljbVZzWVhraU9pSm1ZV3h6ClpTSjlmVEFLQmdncWhrak9QUVFEQWdOSEFEQkVBaUI5Z0lmWGUyOEZZZjllV3FkYXFSbGVIRkhEeHM2aXpWeW8KWW1YOGluRzhoUUlnWjRmblZ2bjhQRTg0eEZ5a3didDAyTktsVjdaR0hoMm5Dd3R6NDBxOGduQT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
	// id_bytes_OrdererMSP := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNmekNDQWlhZ0F3SUJBZ0lRYlk2dFpCWkFPUTRvcVJVY2JaYjVuREFLQmdncWhrak9QUVFEQWpCcE1Rc3cKQ1FZRFZRUUdFd0pWVXpFVE1CRUdBMVVFQ0JNS1EyRnNhV1p2Y201cFlURVdNQlFHQTFVRUJ4TU5VMkZ1SUVaeQpZVzVqYVhOamJ6RVVNQklHQTFVRUNoTUxaWGhoYlhCc1pTNWpiMjB4RnpBVkJnTlZCQU1URG1OaExtVjRZVzF3CmJHVXVZMjl0TUI0WERUSXpNRFV4TmpFeU16WXdNRm9YRFRNek1EVXhNekV5TXpZd01Gb3dXREVMTUFrR0ExVUUKQmhNQ1ZWTXhFekFSQmdOVkJBZ1RDa05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNURFZOaGJpQkdjbUZ1WTJsegpZMjh4SERBYUJnTlZCQU1URTI5eVpHVnlaWEl1WlhoaGJYQnNaUzVqYjIwd1dUQVRCZ2NxaGtqT1BRSUJCZ2dxCmhrak9QUU1CQndOQ0FBUy9vajNZZGNacEFVU0hsaEZ3cVZSblZndDVUYTRLTTdKUHdnTjRWRW85RUNiZXQyOSsKUWk0RzdLZzNtUnU2VjUrbG0rLzMyVThxaVNDQjZHRzFScDB3bzRIQU1JRzlNQTRHQTFVZER3RUIvd1FFQXdJSApnREFNQmdOVkhSTUJBZjhFQWpBQU1Dc0dBMVVkSXdRa01DS0FJS3grYXZ2WVB3dmh2QXJKQldtUGFBb09ac3RQCmdUM2RIdjM4b3RDVnZVdkhNSEFHQ0NvREJBVUdCd2dCQkdSN0ltRjBkSEp6SWpwN0ltaG1Ma0ZtWm1sc2FXRjAKYVc5dUlqb2lJaXdpYUdZdVJXNXliMnhzYldWdWRFbEVJam9pYjNKa1pYSmxjaTVsZUdGdGNHeGxMbU52YlNJcwpJbWhtTGxSNWNHVWlPaUlpTENKeVpXeGhlU0k2SW1aaGJITmxJbjE5TUFvR0NDcUdTTTQ5QkFNQ0EwY0FNRVFDCklGN1lzMVpNaWtiMnVnZVlOR2xuYXRMQjd6Q3FDK3lzN2s1NVFHb01FV1A3QWlBMU5WYUNRSHRPUkJrOEpBK3AKZDBMRGhWT1pOa1NIelFDM2FXMkc2c1pvZ0E9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	// id_bytes, _ := base64.StdEncoding.DecodeString(id_bytes_OrdererMSP)

	identityhex := "0a0a4f7264657265724d535012a0072d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494943667a4343416961674177494241674951625936745a425a414f51346f71525563625a62356e44414b42676771686b6a4f50515144416a42704d5173770a435159445651514745774a56557a45544d4245474131554543424d4b5132467361575a76636d3570595445574d4251474131554542784d4e5532467549455a790a5957356a61584e6a627a45554d4249474131554543684d4c5a586868625842735a53356a62323078467a415642674e5642414d54446d4e684c6d5634595731770a62475575593239744d4234584454497a4d4455784e6a45794d7a59774d466f5844544d7a4d4455784d7a45794d7a59774d466f775744454c4d416b47413155450a42684d4356564d78457a415242674e5642416754436b4e6862476c6d62334a7561574578466a415542674e564241635444564e6862694247636d467559326c7a0a593238784844416142674e5642414d54453239795a4756795a5849755a586868625842735a53356a623230775754415442676371686b6a4f50514942426767710a686b6a4f50514d4242774e434141532f6f6a335964635a70415553486c6846777156526e566774355461344b4d374a5077674e3456456f39454362657432392b0a51693447374b67336d52753656352b6c6d2b2f33325538716953434236474731527030776f3448414d4947394d41344741315564447745422f775145417749480a6744414d42674e5648524d4241663845416a41414d437347413155644977516b4d434b41494b782b61767659507776687641724a42576d5061416f4f5a7374500a67543364487633386f744356765576484d48414743436f44424155474277674242475237496d463064484a7a496a7037496d686d4c6b466d5a6d6c73615746300a61573975496a6f69496977696147597552573579623278736257567564456c45496a6f6962334a6b5a584a6c6369356c654746746347786c4c6d4e76625349730a496d686d4c6c5235634755694f6949694c434a795a57786865534936496d5a6862484e6c496e31394d416f4743437147534d343942414d43413063414d4551430a4946375973315a4d696b6232756765594e476c6e61744c42377a4371432b7973376b353551476f4d45575037416941314e5661435148744f52426b384a412b700a64304c4468564f5a4e6b53487a5143336157324736735a6f67413d3d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a"
	identity, err := hex.DecodeString(identityhex)

	msghex := "0a0208010aaf070a0a4f7264657265724d535012a0072d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494943667a4343416961674177494241674951625936745a425a414f51346f71525563625a62356e44414b42676771686b6a4f50515144416a42704d5173770a435159445651514745774a56557a45544d4245474131554543424d4b5132467361575a76636d3570595445574d4251474131554542784d4e5532467549455a790a5957356a61584e6a627a45554d4249474131554543684d4c5a586868625842735a53356a62323078467a415642674e5642414d54446d4e684c6d5634595731770a62475575593239744d4234584454497a4d4455784e6a45794d7a59774d466f5844544d7a4d4455784d7a45794d7a59774d466f775744454c4d416b47413155450a42684d4356564d78457a415242674e5642416754436b4e6862476c6d62334a7561574578466a415542674e564241635444564e6862694247636d467559326c7a0a593238784844416142674e5642414d54453239795a4756795a5849755a586868625842735a53356a623230775754415442676371686b6a4f50514942426767710a686b6a4f50514d4242774e434141532f6f6a335964635a70415553486c6846777156526e566774355461344b4d374a5077674e3456456f39454362657432392b0a51693447374b67336d52753656352b6c6d2b2f33325538716953434236474731527030776f3448414d4947394d41344741315564447745422f775145417749480a6744414d42674e5648524d4241663845416a41414d437347413155644977516b4d434b41494b782b61767659507776687641724a42576d5061416f4f5a7374500a67543364487633386f744356765576484d48414743436f44424155474277674242475237496d463064484a7a496a7037496d686d4c6b466d5a6d6c73615746300a61573975496a6f69496977696147597552573579623278736257567564456c45496a6f6962334a6b5a584a6c6369356c654746746347786c4c6d4e76625349730a496d686d4c6c5235634755694f6949694c434a795a57786865534936496d5a6862484e6c496e31394d416f4743437147534d343942414d43413063414d4551430a4946375973315a4d696b6232756765594e476c6e61744c42377a4371432b7973376b353551476f4d45575037416941314e5661435148744f52426b384a412b700a64304c4468564f5a4e6b53487a5143336157324736735a6f67413d3d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a1218ee9511ac8c198738e756ec9a1a1e96a21d1d7eb4cf119e873047020101042031081197685be46dbf6e8247a3e3390ff999891279162be87ba64377eb7ff7da042096188dae97aad30a92220ab4923eb7b3b1be7cc4f92872153679922d13ea84e7"
	msg, err := hex.DecodeString(msghex)

	policyManagerGetter := &mocks.ChannelPolicyManagerGetterWithManager{
		Managers: map[string]policies.Manager{
			"testchannel": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: identity, Msg: msg, Mock: mock.Mock{}}},
			},
		},
	}
	// policyManagerGetter.Managers["testchannel"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = []byte("msg1")

	err = VerifyBlock(policyManagerGetter, []byte("testchannel"), 2, block)
	if err != nil {
		fmt.Println(err)
	}
	require.NoError(t, err)

	fmt.Println("----------------")

	// validator, err := createValidator(t)
	// require.NoError(t, err)
	// err = validator.Validate(block)
	// require.NoError(t, err)
}

func createValidator(t *testing.T) (*txvalidator.ValidationRouter, error) {
	peerInstance, cleanup := peer.NewTestPeer2(t)
	defer cleanup()

	var initArg string
	peerInstance.Initialize(
		func(cid string) { initArg = cid },
		nil,
		plugin.MapBasedMapper(map[string]validation.PluginFactory{}),
		&ledgermocks.DeployedChaincodeInfoProvider{},
		nil,
		nil,
		runtime.NumCPU(),
	)

	// testChannelID := fmt.Sprintf("mytestchannelid-%d", rand.Int())
	testChannelID := "testchannel"
	block, err := configtxtest.MakeGenesisBlock(testChannelID)
	if err != nil {
		fmt.Printf("Failed to create a config block, %s err %s\n,", initArg, err)
		t.FailNow()
	}

	err = peerInstance.CreateChannel(testChannelID, block, &ledgermocks.DeployedChaincodeInfoProvider{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create chain %s", err)
	}

	validator, err := CreateTxValidatorViaPeer(peerInstance, testChannelID, nil, nil)

	return validator, err
}
