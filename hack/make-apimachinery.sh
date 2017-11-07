#!/bin/bash

# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build apimachinery executables from vendor-ed dependencies

WORK_DIR=`mktemp -d`

function cleanup {
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

mkdir -p ${WORK_DIR}/go/
ln -s ${GOPATH}/src/github.com/justinsb/scaler/vendor/ ${WORK_DIR}/go/src

GOPATH=${WORK_DIR}/go/ go install k8s.io/code-generator/cmd/conversion-gen/
cp ${WORK_DIR}/go/bin/conversion-gen ${GOPATH}/bin/

GOPATH=${WORK_DIR}/go/ go install k8s.io/code-generator/cmd/deepcopy-gen/
cp ${WORK_DIR}/go/bin/deepcopy-gen ${GOPATH}/bin/

GOPATH=${WORK_DIR}/go/ go install k8s.io/code-generator/cmd/defaulter-gen/
cp ${WORK_DIR}/go/bin/defaulter-gen ${GOPATH}/bin/

GOPATH=${WORK_DIR}/go/ go install k8s.io/code-generator/cmd/client-gen/
cp ${WORK_DIR}/go/bin/client-gen ${GOPATH}/bin/

GOPATH=${WORK_DIR}/go/ go install k8s.io/code-generator/cmd/lister-gen/
cp ${WORK_DIR}/go/bin/lister-gen ${GOPATH}/bin/

GOPATH=${WORK_DIR}/go/ go install k8s.io/code-generator/cmd/informer-gen/
cp ${WORK_DIR}/go/bin/informer-gen ${GOPATH}/bin/

