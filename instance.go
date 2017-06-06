package main

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type instanceController struct {
	client     kubernetes.Interface
	tprclient  *rest.RESTClient
	nodeLister storeToNodeLister
	controller cache.ControllerInterface
}

func serviceInstanceController(client kubernetes.Interface, tprclient *rest.RESTClient) *instanceController {
	ic := &instanceController{
		client:    client,
		tprclient: tprclient,
	}

	instancelistwatcher := cache.NewListWatchFromClient(tprclient, SERVICE_INSTANCES, "", nil)

	store, controller := cache.NewInformer(
		instancelistwatcher,
		// The types of objects this informer will return
		&ServiceInstance{},
		// The resync period of this object. This will force a re-queue of all cached objects at this interval.
		// Every object will trigger the `Updatefunc` even if there have been no actual updates triggered.
		// In some cases you can set this to a very high interval - as you can assume you will see periodic updates in normal operation.
		// The interval is set low here for demo purposes.
		120*time.Second,
		// Callback Functions to trigger on add/update/delete
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ic.addHandler,
			UpdateFunc: func(old, new interface{}) { ic.updateHandler(old, new) },
			DeleteFunc: ic.deleteHandler,
		},
	)

	ic.controller = controller
	// Convert the cache.Store to a nodeLister to avoid some boilerplate (e.g. convert runtime.Objects to *ServiceInstance)
	ic.nodeLister = storeToNodeLister{store}

	return ic
}

func (c *instanceController) addHandler(obj interface{}) {
	instance := obj.(*ServiceInstance)
	glog.V(4).Infof("instance %v ADDED to %v namespace.", instance.Metadata.Name, instance.Metadata.Namespace)
	return
}

func (c *instanceController) updateHandler(old, new interface{}) {
	instance := new.(*ServiceInstance)
	oldInstance := old.(*ServiceInstance)
	glog.V(4).Info("instance UPDATED,", instance.Metadata.ResourceVersion, instance.Spec, oldInstance.Metadata.ResourceVersion, oldInstance.Spec)
	if instance == oldInstance {
		glog.V(4).Info("nothing changes.")
	} else {
		glog.V(4).Info("something changes.")
	}
	return
}

func (c *instanceController) deleteHandler(obj interface{}) {
	instance := obj.(*ServiceInstance)
	glog.V(4).Infof("instance DELETED %v", instance)
	return
}
