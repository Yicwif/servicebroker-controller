package main

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type brokerController struct {
	client     kubernetes.Interface
	tprclient  *rest.RESTClient
	nodeLister storeToNodeLister
	controller cache.ControllerInterface
}

func serviceBrokerController(client kubernetes.Interface, tprclient *rest.RESTClient) *brokerController {
	broker := &brokerController{
		client:    client,
		tprclient: tprclient,
	}

	brokerlistwatcher := cache.NewListWatchFromClient(tprclient, SERVICE_BROKERS, api.NamespaceDefault, nil)

	store, controller := cache.NewInformer(
		brokerlistwatcher,
		// The types of objects this informer will return
		// &v1.Pod{},
		&ServiceBroker{},
		// The resync period of this object. This will force a re-queue of all cached objects at this interval.
		// Every object will trigger the `Updatefunc` even if there have been no actual updates triggered.
		// In some cases you can set this to a very high interval - as you can assume you will see periodic updates in normal operation.
		// The interval is set low here for demo purposes.
		300*time.Second,
		// Callback Functions to trigger on add/update/delete
		cache.ResourceEventHandlerFuncs{
			AddFunc:    broker.addHandler,
			UpdateFunc: func(old, new interface{}) { broker.updateHandler(old, new) },
			DeleteFunc: broker.deleteHandler,
		},
	)

	broker.controller = controller
	// Convert the cache.Store to a nodeLister to avoid some boilerplate (e.g. convert runtime.Objects to *v1.Nodes)
	// TODO: use upstream cache.StoreToNodeLister once v3.0.0 client-go available
	broker.nodeLister = storeToNodeLister{store}

	return broker
}

func (c *brokerController) addHandler(obj interface{}) {
	broker := obj.(*ServiceBroker)
	glog.V(4).Infof("new broker %v added, URL: %v, username: %v, password: %v",
		broker.Metadata.Name, broker.Spec.URL, broker.Spec.Username, broker.Spec.Password)
	return
}

func (c *brokerController) updateHandler(old, new interface{}) {
	broker := new.(*ServiceBroker)
	oldBroker := old.(*ServiceBroker)
	glog.V(4).Infof("broker %v updated, URL: %v, username: %v, password: %v old broker: %v URL: %v, username: %v, password: %v",
		broker.Metadata.Name, broker.Spec.URL, broker.Spec.Username, broker.Spec.Password,
		oldBroker.Metadata.Name, oldBroker.Spec.URL, oldBroker.Spec.Username, oldBroker.Spec.Password)
	return
}

func (c *brokerController) deleteHandler(obj interface{}) {
	broker := obj.(*ServiceBroker)
	glog.V(4).Infof("broker %v deleted", broker.Metadata.Name, broker.Spec.URL, broker.Spec.Username, broker.Spec.Password)
	return
}
