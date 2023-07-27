# Copyright IBM Corp. All Rights Reserved.
# Copyright 2020 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Description:
#   Sets up the template of a docker environment to run FPC go chaincode
#
#  Configuration (build) parameters (for defaults, see below section with ARGs)
#  - fpc image version:          FPC_VERSION
#  - sgxmode:                    SGX_MODE
#  - Chaincode as a Server Port: CAAS_PORT

ARG FPC_CCENV_IMAGE=hyperledger/fabric-private-chaincode-ccenv:main

FROM ${FPC_CCENV_IMAGE}

ARG SGX_MODE
ENV SGX_MODE=${SGX_MODE}
ARG OE_SIMULATION
ENV OE_SIMULATION=${OE_SIMULATION}

ENV MSP_PATH="/project/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/"
COPY msp ${MSP_PATH}

ARG CAAS_PORT=50051
ARG CC_PATH="/usr/local/bin"

ENV CORE_CONFIG_PATH="/usr/local/bin/core.yaml"
COPY core.yaml ${CORE_CONFIG_PATH}

ENV ORDERER_CONFIC_PATH="/usr/local/bin/orderer.yaml"
COPY orderer.yaml ${ORDERER_CONFIC_PATH}

ENV DEV_PATH="/usr/local/bin/sampleconfig"
COPY sample-fabric-config ${DEV_PATH}

ENV BLOCK_PATH="/usr/local/bin/tmpBlocks/"
COPY tmpBlocks ${BLOCK_PATH}

COPY tle-bundle ${CC_PATH}/tle
COPY client ${CC_PATH}/client

WORKDIR ${CC_PATH}
EXPOSE ${CAAS_PORT}
EXPOSE 50051

CMD ["tle"]
# CMD ["sleep", "infinity"]
