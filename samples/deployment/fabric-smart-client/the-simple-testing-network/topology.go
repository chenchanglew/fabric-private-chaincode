/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
)

const (
	defaultChaincodeName      = "kv-test-go"
	defaultChaincodeImageName = "fpc/fpc-kv-test-go"
	defaultChaincodeImageTag  = "latest"
	defaultChaincodeMRENCLAVE = "fakeMRENCLAVE"
	defaultLoggingSpec        = "info"
)

func Fabric() []api.Topology {
	config := setup()

	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetLogging(config.loggingSpec, "")

	// in this example we use the FPC kv-test-go chaincode
	// we just need to set the docker images
	fabricTopology.EnableFPC()
	fabricTopology.AddFPC(config.chaincodeName, config.chaincodeImage, config.fpcOptions...)

	fabricTopology.Templates = &topology.Templates{ConfigTx: ModifyConfigTxTemplate, Core: ModifyCoreTemplate}

	// bring hyperledger explorer into the game
	// you can reach it http://localhost:8080 with admin:admin
	// monitoringTopology := monitoring.NewTopology()
	// monitoringTopology.EnableHyperledgerExplorer()

	// return []api.Topology{fabricTopology, fsc.NewTopology(), monitoringTopology}
	return []api.Topology{fabricTopology, fsc.NewTopology()}
}

type config struct {
	loggingSpec    string
	chaincodeName  string
	chaincodeImage string
	fpcOptions     []func(chaincode *topology.ChannelChaincode)
}

// setup prepares a config helper struct, containing some additional configuration that can be injected via environment variables
func setup() *config {
	config := &config{}

	// export FABRIC_LOGGING_SPECS=info
	config.loggingSpec = os.Getenv("FABRIC_LOGGING_SPEC")
	if len(config.loggingSpec) == 0 {
		config.loggingSpec = defaultLoggingSpec
	}

	// export CC_NAME=kv-test-go
	config.chaincodeName = os.Getenv("CC_NAME")
	if len(config.chaincodeName) == 0 {
		config.chaincodeName = defaultChaincodeName
	}

	// export FPC_CHAINCODE_IMAGE=fpc/fpc-kv-test-go:latest
	config.chaincodeImage = os.Getenv("FPC_CHAINCODE_IMAGE")
	if len(config.chaincodeImage) == 0 {
		config.chaincodeImage = fmt.Sprintf("%s:%s", defaultChaincodeImageName, defaultChaincodeImageTag)
	}

	// get mrenclave
	mrenclave := os.Getenv("FPC_MRENCLAVE")
	if len(mrenclave) == 0 {
		mrenclave = defaultChaincodeMRENCLAVE
	}
	config.fpcOptions = append(config.fpcOptions, topology.WithMREnclave(mrenclave))

	// check if we are running in SGX HW mode
	// export SGX_MODE=SIM
	if strings.ToUpper(os.Getenv("SGX_MODE")) == "HW" {
		sgxDevicePath := DetectSgxDevicePath()
		config.fpcOptions = append(config.fpcOptions, topology.WithSGXDevicesPaths(sgxDevicePath))
	}

	return config
}

func DetectSgxDevicePath() []string {
	possiblePaths := []string{"/dev/isgx", "/dev/sgx/enclave"}
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err != nil {
			continue
		} else {
			// first found path returns
			return []string{p}
		}
	}

	panic("no sgx device path found")
}

func ReadMrenclaveFromFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("cannot read mrenclave from %s", path))
	}

	mrenclave := strings.TrimSpace(string(data))
	if len(mrenclave) == 0 {
		panic(fmt.Errorf("mrenclave file empty"))
	}

	return mrenclave
}

const ModifyConfigTxTemplate = `---
{{ with $w := . -}}
Organizations:{{ range .PeerOrgs }}
- &{{ .MSPID }}
  Name: {{ .Name }}
  ID: {{ .MSPID }}
  MSPDir: {{ $w.PeerOrgMSPDir . }}
  Policies:
    {{- if .EnableNodeOUs }}
    Readers:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin', '{{.MSPID}}.peer', '{{.MSPID}}.client')
    Writers:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin', '{{.MSPID}}.client')
    Endorsement:
      Type: Signature
      Rule: OR('{{.MSPID}}.peer')
    Admins:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin')
    {{- else }}
    Readers:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Writers:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Endorsement:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Admins:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin')
    {{- end }}
  AnchorPeers:{{ range $w.AnchorsInOrg .Name }}
  - Host: 127.0.0.1
    Port: {{ $w.PeerPort . "Listen" }}
  {{- end }}
{{- end }}
{{- range .IdemixOrgs }}
- &{{ .MSPID }}
  Name: {{ .Name }}
  ID: {{ .MSPID }}
  MSPDir: {{ $w.IdemixOrgMSPDir . }}
  MSPType: idemix
  Policies:
    {{- if .EnableNodeOUs }}
    Readers:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin', '{{.MSPID}}.peer', '{{.MSPID}}.client')
    Writers:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin', '{{.MSPID}}.client')
    Endorsement:
      Type: Signature
      Rule: OR('{{.MSPID}}.peer')
    Admins:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin')
    {{- else }}
    Readers:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Writers:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Endorsement:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Admins:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin')
    {{- end }}
{{ end }}
{{- range .OrdererOrgs }}
- &{{ .MSPID }}
  Name: {{ .Name }}
  ID: {{ .MSPID }}
  MSPDir: {{ $w.OrdererOrgMSPDir . }}
  Policies:
  {{- if .EnableNodeOUs }}
    Readers:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin', '{{.MSPID}}.orderer', '{{.MSPID}}.client')
    Writers:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin', '{{.MSPID}}.orderer', '{{.MSPID}}.client')
    Admins:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin')
  {{- else }}
    Readers:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Writers:
      Type: Signature
      Rule: OR('{{.MSPID}}.member')
    Admins:
      Type: Signature
      Rule: OR('{{.MSPID}}.admin')
  {{- end }}
  OrdererEndpoints:{{ range $w.OrderersInOrg .Name }}
  - 127.0.0.1:{{ $w.OrdererPort . "Listen" }}
  {{- end }}
{{ end }}

Channel: &ChannelDefaults
  Capabilities:
    V2_0: true
  Policies: &DefaultPolicies
    Readers:
      Type: ImplicitMeta
      Rule: ANY Readers
    Writers:
      Type: ImplicitMeta
      Rule: ANY Writers
    Admins:
      Type: ImplicitMeta
      Rule: MAJORITY Admins

Profiles:{{ range .Profiles }}
  {{ .Name }}:
    {{- if .ChannelCapabilities}}
    Capabilities:{{ range .ChannelCapabilities}}
      {{ . }}: true
    {{- end}}
    Policies:
      <<: *DefaultPolicies
    {{- else }}
    <<: *ChannelDefaults
    {{- end}}
    {{- if .Orderers }}
    Orderer:
      OrdererType: {{ $w.Consensus.Type }}
      Addresses:{{ range .Orderers }}{{ with $w.Orderer . }}
      - 127.0.0.1:{{ $w.OrdererPort . "Listen" }}
      {{- end }}{{ end }}
      BatchTimeout: 1s
      BatchSize:
        MaxMessageCount: 2
        AbsoluteMaxBytes: 98 MB
        PreferredMaxBytes: 4096 KB
      Capabilities:
        V2_0: true
      {{- if eq $w.Consensus.Type "kafka" }}
      Kafka:
        Brokers:{{ range $w.BrokerAddresses "HostPort" }}
        - {{ . }}
        {{- end }}
      {{- end }}
      {{- if eq $w.Consensus.Type "etcdraft" }}
      EtcdRaft:
        Options:
          TickInterval: 500ms
          SnapshotIntervalSize: 1 KB
        Consenters:{{ range .Orderers }}{{ with $w.Orderer . }}
        - Host: 127.0.0.1
          Port: {{ $w.OrdererPort . "Cluster" }}
          ClientTLSCert: {{ $w.OrdererLocalCryptoDir . "tls" }}/server.crt
          ServerTLSCert: {{ $w.OrdererLocalCryptoDir . "tls" }}/server.crt
        {{- end }}{{- end }}
      {{- end }}
      Organizations:{{ range $w.OrgsForOrderers .Orderers }}
      - *{{ .MSPID }}
      {{- end }}
      Policies:
        Readers:
          Type: ImplicitMeta
          Rule: ANY Readers
        Writers:
          Type: ImplicitMeta
          Rule: ANY Writers
        Admins:
          Type: ImplicitMeta
          Rule: MAJORITY Admins
        BlockValidation:
          Type: ImplicitMeta
          Rule: ANY Writers
    {{- end }}
    {{- if .Consortium }}
    Consortium: {{ .Consortium }}
    Application:
      Capabilities:
      {{- if .AppCapabilities }}{{ range .AppCapabilities }}
        {{ . }}: true
        {{- end }}
      {{- else }}
        V2_0: true
      {{- end }}
      Organizations:{{ range .Organizations }}
      - *{{ ($w.Organization .).MSPID }}
      {{- end}}
      Policies:
      {{- if .Policies }}{{ range .Policies }} 
        {{ .Name }}:
          Type: {{ .Type }}
          Rule: {{ .Rule }}
      {{- end }}
      {{- else }}
        Readers:
          Type: ImplicitMeta
          Rule: ANY Readers
        Writers:
          Type: ImplicitMeta
          Rule: ANY Writers
        Admins:
          Type: ImplicitMeta
          Rule: MAJORITY Admins
        LifecycleEndorsement:
          Type: ImplicitMeta
          Rule: "MAJORITY Endorsement"
        Endorsement:
          Type: ImplicitMeta
          Rule: "MAJORITY Endorsement"
      {{- end }}
    {{- else }}
    Consortiums:{{ range $w.Consortiums }}
      {{ .Name }}:
        Organizations:{{ range .Organizations }}
        - *{{ ($w.Organization .).MSPID }}
        {{- end }}
    {{- end }}
    {{- end }}
{{- end }}
{{ end }}
`

const ModifyCoreTemplate = `---
logging:
  spec: {{ .Logging.Spec }} 
  format: {{ .Logging.Format }}

peer:
  id: {{ Peer.ID }}
  networkId: {{ .NetworkID }}
  address: 127.0.0.1:{{ .PeerPort Peer "Listen" }}
  addressAutoDetect: true
  listenAddress: 127.0.0.1:{{ .PeerPort Peer "Listen" }}
  chaincodeListenAddress: 127.0.0.1:{{ .PeerPort Peer "Chaincode" }}
  keepalive:
    minInterval: 60s
    interval: 300s
    timeout: 600s
    client:
      interval: 60s
      timeout: 600s
    deliveryClient:
      interval: 60s
      timeout: 20s
  gossip:
    bootstrap: 127.0.0.1:{{ .PeerPort Peer "Listen" }}
    endpoint: 127.0.0.1:{{ .PeerPort Peer "Listen" }}
    externalEndpoint: 127.0.0.1:{{ .PeerPort Peer "Listen" }}
    useLeaderElection: true
    orgLeader: false
    membershipTrackerInterval: 5s
    maxBlockCountToStore: 100
    maxPropagationBurstLatency: 10ms
    maxPropagationBurstSize: 10
    propagateIterations: 1
    propagatePeerNum: 3
    pullInterval: 4s
    pullPeerNum: 3
    requestStateInfoInterval: 4s
    publishStateInfoInterval: 4s
    stateInfoRetentionInterval:
    publishCertPeriod: 10s
    dialTimeout: 3s
    connTimeout: 2s
    recvBuffSize: 20
    sendBuffSize: 200
    digestWaitTime: 1s
    requestWaitTime: 1500ms
    responseWaitTime: 2s
    aliveTimeInterval: 5s
    aliveExpirationTimeout: 25s
    reconnectInterval: 25s
    election:
      startupGracePeriod: 15s
      membershipSampleInterval: 1s
      leaderAliveThreshold: 10s
      leaderElectionDuration: 5s
    pvtData:
      pullRetryThreshold: 7s
      transientstoreMaxBlockRetention: 1000
      pushAckTimeout: 3s
      btlPullMargin: 10
      reconcileBatchSize: 10
      reconcileSleepInterval: 10s
      reconciliationEnabled: true
      skipPullingInvalidTransactionsDuringCommit: false
    state:
       enabled: true
       checkInterval: 10s
       responseTimeout: 3s
       batchSize: 10
       blockBufferSize: 100
       maxRetries: 3
  events:
    address: 127.0.0.1:{{ .PeerPort Peer "Events" }}
    buffersize: 100
    timeout: 10ms
    timewindow: 15m
    keepalive:
      minInterval: 60s
  tls:
    enabled:  true
    clientAuthRequired: {{ .ClientAuthRequired }}
    cert:
      file: {{ .PeerLocalTLSDir Peer }}/server.crt
    key:
      file: {{ .PeerLocalTLSDir Peer }}/server.key
    clientCert:
      file: {{ .PeerLocalTLSDir Peer }}/server.crt
    clientKey:
      file: {{ .PeerLocalTLSDir Peer }}/server.key
    rootcert:
      file: {{ .PeerLocalTLSDir Peer }}/ca.crt
    clientRootCAs:
      files:
      - {{ .PeerLocalTLSDir Peer }}/ca.crt
  authentication:
    timewindow: 15m
  fileSystemPath: filesystem
  BCCSP:
    Default: SW
    SW:
      Hash: SHA2
      Security: 256
      FileKeyStore:
        KeyStore:
  mspConfigPath: {{ .PeerLocalMSPDir Peer }}
  localMspId: {{ (.Organization Peer.Organization).MSPID }}
  deliveryclient:
    reconnectTotalTimeThreshold: 3600s
  localMspType: bccsp
  profile:
    enabled:     false
    listenAddress: 127.0.0.1:{{ .PeerPort Peer "ProfilePort" }}
  handlers:
    authFilters:
    - name: DefaultAuth
    - name: ExpirationCheck
    decorators:
    - name: DefaultDecorator
    endorsers:
      escc:
        name: DefaultEndorsement
    validators:
      vscc:
        name: DefaultValidation
      {{ if .PvtTxSupport }}vscc_pvt:
        name: DefaultPvtValidation
        library: {{ end }}
      {{ if .MSPvtTxSupport }}vscc_mspvt:
        name: DefaultMSPvtValidation
        library: {{ end }}
      {{ if .FabTokenSupport }}vscc_token:
        name: DefaultTokenValidation
        library: {{ end }}
  validatorPoolSize:
  discovery:
    enabled: true
    authCacheEnabled: true
    authCacheMaxSize: 1000
    authCachePurgeRetentionRatio: 0.75
    orgMembersAllowedAccess: false
  limits:
    concurrency:
      qscc: 500

vm:
  endpoint: unix:///var/run/docker.sock
  docker:
    tls:
      enabled: false
      ca:
        file: docker/ca.crt
      cert:
        file: docker/tls.crt
      key:
        file: docker/tls.key
    attachStdout: true
    hostConfig:
      NetworkMode: host
      LogConfig:
        Type: json-file
        Config:
          max-size: "50m"
          max-file: "5"
      Memory: 2147483648

chaincode:
  builder: $(DOCKER_NS)/fabric-ccenv:$(PROJECT_VERSION)
  pull: false
  golang:
    runtime: $(DOCKER_NS)/fabric-baseos:$(PROJECT_VERSION)
    dynamicLink: false
  car:
    runtime: $(DOCKER_NS)/fabric-baseos:$(PROJECT_VERSION)
  java:
    runtime: $(DOCKER_NS)/fabric-javaenv:latest
  node:
    runtime: $(DOCKER_NS)/fabric-nodeenv:latest
  installTimeout: 1200s
  startuptimeout: 1200s
  executetimeout: 1200s
  mode: net
  keepalive: 0
  system:
    _lifecycle: enable
    cscc:       enable
    lscc:       enable
    qscc:       enable
  logging:
    level:  debug
    shim:   debug
    format: '%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'
  externalBuilders: {{ range .ExternalBuilders }}
    - path: {{ .Path }}
      name: {{ .Name }}
      propagateEnvironment: {{ range .PropagateEnvironment }}
         - {{ . }}
      {{- end }}
  {{- end }}

ledger:
  blockchain:
  state:
    stateDatabase: goleveldb
    couchDBConfig:
      couchDBAddress: 127.0.0.1:5984
      username:
      password:
      maxRetries: 3
      maxRetriesOnStartup: 10
      requestTimeout: 35s
      queryLimit: 10000
      maxBatchUpdateSize: 1000
      warmIndexesAfterNBlocks: 1
  history:
    enableHistoryDatabase: true

operations:
  listenAddress: 127.0.0.1:{{ .PeerPort Peer "Operations" }}
  tls:
    enabled: false
    cert:
      file: {{ .PeerLocalTLSDir Peer }}/server.crt
    key:
      file: {{ .PeerLocalTLSDir Peer }}/server.key
    clientAuthRequired: {{ .ClientAuthRequired }}
    clientRootCAs:
      files:
      - {{ .PeerLocalTLSDir Peer }}/ca.crt
metrics:
  provider: {{ .MetricsProvider }}
  statsd:
    network: udp
    address: {{ if .StatsdEndpoint }}{{ .StatsdEndpoint }}{{ else }}127.0.0.1:8125{{ end }}
    writeInterval: 5s
    prefix: {{ ReplaceAll (ToLower Peer.ID) "." "_" }}
`
