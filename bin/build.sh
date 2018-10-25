#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="holepunch-server"
BINTRAY_PROJECT="function61/holepunch-server"
COMPILE_IN_DIRECTORY="cmd/holepunch-server"

standardBuildProcess
