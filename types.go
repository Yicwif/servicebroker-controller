/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"encoding/json"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
)

type ServiceBrokerSpec struct {
	Foo string `json:"foo"`
	Bar bool   `json:"bar"`
}

type ServiceBroker struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec ServiceBrokerSpec `json:"spec"`
}

type ServiceBrokerList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []ServiceBroker `json:"items"`
}

// Required to satisfy Object interface
func (sb *ServiceBroker) GetObjectKind() unversioned.ObjectKind {
	return &sb.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (sb *ServiceBroker) GetObjectMeta() meta.Object {
	return &sb.Metadata
}

// Required to satisfy Object interface
func (sbl *ServiceBrokerList) GetObjectKind() unversioned.ObjectKind {
	return &sbl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (sbl *ServiceBrokerList) GetListMeta() unversioned.List {
	return &sbl.Metadata
}

// The code below is used only to work around a known problem with third-party
// resources and ugorji. If/when these issues are resolved, the code below
// should no longer be required.

type ServiceBrokerListCopy ServiceBrokerList
type ServiceBrokerCopy ServiceBroker

func (sb *ServiceBroker) UnmarshalJSON(data []byte) error {
	tmp := ServiceBrokerCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ServiceBroker(tmp)
	*sb = tmp2
	return nil
}

func (sbl *ServiceBrokerList) UnmarshalJSON(data []byte) error {
	tmp := ServiceBrokerListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ServiceBrokerList(tmp)
	*sbl = tmp2
	return nil
}
