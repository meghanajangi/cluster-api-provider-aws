/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Run go generate to regenerate this mock.
//go:generate ../../../../../hack/tools/bin/mockgen -destination ssmapi_mock.go -package mock_ssmiface github.com/aws/aws-sdk-go/service/ssm/ssmiface SSMAPI
//go:generate /usr/bin/env bash -c "cat ../../../../../hack/boilerplate/boilerplate.generatego.txt ssmapi_mock.go > _ssmapi_mock.go && mv _ssmapi_mock.go ssmapi_mock.go"

package mock_ssmiface // nolint:stylecheck
