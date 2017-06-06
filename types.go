package main

import (
	"encoding/json"

	"fmt"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
)

const (
	TPR_SERVICE_BROKER   = "service-broker"
	TPR_BACKING_SERVICE  = "backing-service"
	TPR_SERVICE_INSTANCE = "service-instance"
	TPR_GROUP            = "datafoundry.io"
	TPR_VERSION          = "v1"
	SERVICE_BROKERS      = "servicebrokers"
	SERVICE_INSTANCES    = "serviceinstances"
	BACKING_SERVICES     = "backingservices"
)

var (
	TPRKinds = []string{TPR_SERVICE_BROKER, TPR_BACKING_SERVICE, TPR_SERVICE_INSTANCE}
	TPRDesc  = map[string]string{
		TPR_SERVICE_BROKER:   "ServiceBroker agent on DataFoundry",
		TPR_BACKING_SERVICE:  "Service catalog from a ServiceBroker",
		TPR_SERVICE_INSTANCE: "BackingService instance",
	}
)

func tprName(kind string) string {
	return fmt.Sprintf("%s.%s", kind, TPR_GROUP)
}

type ServiceBrokerSpec struct {
	URL        string `json:"url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	APIVersion string `json:"api-version,omitempty"`
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

type ServiceInstanceSpec map[string]interface{}

type ServiceInstance struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec ServiceInstanceSpec `json:"spec"`
}

type ServiceInstanceList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []ServiceInstance `json:"items"`
}

type BackingServiceSpec map[string]interface{}

type BackingService struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec BackingServiceSpec `json:"spec"`
}

type BackingServiceList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []BackingService `json:"items"`
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
func (sblist *ServiceBrokerList) GetObjectKind() unversioned.ObjectKind {
	return &sblist.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (sblist *ServiceBrokerList) GetListMeta() unversioned.List {
	return &sblist.Metadata
}

// Required to satisfy Object interface
func (bsi *ServiceInstance) GetObjectKind() unversioned.ObjectKind {
	return &bsi.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (bsi *ServiceInstance) GetObjectMeta() meta.Object {
	return &bsi.Metadata
}

// Required to satisfy Object interface
func (bsilist *ServiceInstanceList) GetObjectKind() unversioned.ObjectKind {
	return &bsilist.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (bsilist *ServiceInstanceList) GetListMeta() unversioned.List {
	return &bsilist.Metadata
}

// Required to satisfy Object interface
func (bs *BackingService) GetObjectKind() unversioned.ObjectKind {
	return &bs.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (bs *BackingService) GetObjectMeta() meta.Object {
	return &bs.Metadata
}

// Required to satisfy Object interface
func (bslist *BackingServiceList) GetObjectKind() unversioned.ObjectKind {
	return &bslist.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (bslist *BackingServiceList) GetListMeta() unversioned.List {
	return &bslist.Metadata
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

func (sblist *ServiceBrokerList) UnmarshalJSON(data []byte) error {
	tmp := ServiceBrokerListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ServiceBrokerList(tmp)
	*sblist = tmp2
	return nil
}

type ServiceInstanceListCopy ServiceInstanceList
type ServiceInstanceCopy ServiceInstance

func (bsi *ServiceInstance) UnmarshalJSON(data []byte) error {
	tmp := ServiceInstanceCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ServiceInstance(tmp)
	*bsi = tmp2
	return nil
}

func (bsilist *ServiceInstanceList) UnmarshalJSON(data []byte) error {
	tmp := ServiceInstanceListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ServiceInstanceList(tmp)
	*bsilist = tmp2
	return nil
}

type BackingServiceListCopy BackingServiceList
type BackingServiceCopy BackingService

func (bs *BackingService) UnmarshalJSON(data []byte) error {
	tmp := BackingServiceCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := BackingService(tmp)
	*bs = tmp2
	return nil
}

func (bslist *BackingServiceList) UnmarshalJSON(data []byte) error {
	tmp := BackingServiceListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := BackingServiceList(tmp)
	*bslist = tmp2
	return nil
}
