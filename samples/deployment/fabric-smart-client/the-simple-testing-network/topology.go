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

	fabricTopology.Templates = &topology.Templates{ConfigTx: ModifyConfigTxTemplate}

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
      BatchTimeout: 2s
      BatchSize:
        MaxMessageCount: 10
        AbsoluteMaxBytes: 98 MB
        PreferredMaxBytes: 2048 KB
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
