package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var Version string = "none" // Set at compile time

func main() {
	// When running as a pod in-cluster, a kubeconfig is not needed. Instead this will make use of the service account injected into the pod.
	// However, allow the use of a local kubeconfig as this can make local development & testing easier.
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")

	// We log to stderr because glog will default to logging to a file.
	// By setting this debugging is easier via `kubectl logs`
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Build the client config - optionally using a provided kubeconfig file.
	config, err := GetClientConfig(*kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to load client config: %v", err)
	}

	// Construct the Kubernetes client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	prepareTPR(client)

	// make a new config for our extension's API group, using the first config as a baseline
	var tprconfig *rest.Config
	tprconfig = config
	configureClient(tprconfig)

	tprclient, err := rest.RESTClientFor(tprconfig)
	if err != nil {
		panic(err)
	}

	glog.Infof("Starting Servicebroker controller, version: %v", Version)
	go newRebootController(client, tprclient).controller.Run(wait.NeverStop)
	select {}
}

func prepareTPR(client kubernetes.Interface) {
	glog.V(4).Info("prepare TPR.")

	for _, kind := range TPRKinds {
		name := tprName(kind)
		result, err := client.Extensions().ThirdPartyResources().Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				tpr := &v1beta1.ThirdPartyResource{
					ObjectMeta: v1.ObjectMeta{
						Name: name,
					},
					Versions: []v1beta1.APIVersion{
						{Name: TPRVersion},
					},
					Description: TPRDesc[kind],
				}

				result, err := client.Extensions().ThirdPartyResources().Create(tpr)
				if err != nil {
					panic(err)
				}
				fmt.Printf("CREATED: %#v\nFROM: %#v\n", result, tpr)
			} else {
				panic(err)
			}
		}
		fmt.Printf("SKIPPING: already exists %#v\n", result)
	}
}

func configureClient(config *rest.Config) {
	groupversion := unversioned.GroupVersion{
		Group:   TPRGroup,
		Version: TPRVersion,
	}

	config.GroupVersion = &groupversion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				groupversion,
				&ServiceBroker{},
				&ServiceBrokerList{},
				&api.ListOptions{},
				&api.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(api.Scheme)
}

type rebootController struct {
	client     kubernetes.Interface
	tprclient  *rest.RESTClient
	nodeLister storeToNodeLister
	controller cache.ControllerInterface
}

func newRebootController(client kubernetes.Interface, tprclient *rest.RESTClient) *rebootController {
	rc := &rebootController{
		client:    client,
		tprclient: tprclient,
	}

	store, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(alo api.ListOptions) (runtime.Object, error) {
				var lo v1.ListOptions
				v1.Convert_api_ListOptions_To_v1_ListOptions(&alo, &lo, nil)

				// We do not add any selectors because we want to watch all nodes.
				// This is so we can determine the total count of "unavailable" nodes.
				// However, this could also be implemented using multiple informers (or better, shared-informers)
				// return client.Core().Pods().List(lo)

				// return client.Core().Pods("").List(lo)

				// return client.CoreV1().RESTClient().Get().Resource("thirdpartyresources").Do().Get()
				return client.Extensions().ThirdPartyResources().List(lo)

			},
			WatchFunc: func(alo api.ListOptions) (watch.Interface, error) {
				var lo v1.ListOptions
				v1.Convert_api_ListOptions_To_v1_ListOptions(&alo, &lo, nil)
				// return client.Core().Pods("").Watch(lo)
				// return client.CoreV1().RESTClient().Get().Resource("thirdpartyresources").Watch()
				return client.Extensions().ThirdPartyResources().Watch(lo)
			},
		},
		// The types of objects this informer will return
		// &v1.Pod{},
		&v1beta1.ThirdPartyResource{},
		// The resync period of this object. This will force a re-queue of all cached objects at this interval.
		// Every object will trigger the `Updatefunc` even if there have been no actual updates triggered.
		// In some cases you can set this to a very high interval - as you can assume you will see periodic updates in normal operation.
		// The interval is set low here for demo purposes.
		10*time.Second,
		// Callback Functions to trigger on add/update/delete
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rc.handler,
			UpdateFunc: func(old, new interface{}) { rc.handler(new) },
			DeleteFunc: rc.handler,
		},
	)

	rc.controller = controller
	// Convert the cache.Store to a nodeLister to avoid some boilerplate (e.g. convert runtime.Objects to *v1.Nodes)
	// TODO(aaron): use upstream cache.StoreToNodeLister once v3.0.0 client-go available
	rc.nodeLister = storeToNodeLister{store}

	return rc
}

func (c *rebootController) handler(obj interface{}) {
	// TODO(aaron): This would be better handled using a workqueue. This will be added to client-go during v1.6.x release.
	//   As we process objects, add to queue for processing, rather than potentially rebooting whichver node checked in last.
	//   A good example of this pattern is shown in: https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md
	//   We could also protect against operating against a partial cache by not starting processing until cached synced.

	// pod := obj.(*v1.Pod)
	// glog.V(4).Infof("Pod: %s, status: %v, namespace: %s", pod.Name, podStatus(pod), pod.Namespace)

	tpr := obj.(*v1beta1.ThirdPartyResource)
	glog.V(4).Infof("TPR: %s, version: %v, desc: %v", tpr.Name, tpr.Versions, tpr.Description)
	sblist := new(ServiceBrokerList)
	err := c.tprclient.Get().Resource("servicebrokers").Namespace(api.NamespaceDefault).Do().Into(sblist)
	if err != nil {
		glog.Error(err)
	} else {
		glog.V(4).Info(sblist)
	}
	func() {
		var example ServiceBroker

		err = c.tprclient.Get().
			Resource("examples").
			Namespace(api.NamespaceDefault).
			Name("example1").
			Do().Into(&example)

		if err != nil {
			if errors.IsNotFound(err) {
				// Create an instance of our TPR
				example := &ServiceBroker{
					Metadata: api.ObjectMeta{
						Name: "example1",
					},
					Spec: ServiceBrokerSpec{
						Foo: "hello",
						Bar: true,
					},
				}

				var result ServiceBroker
				err = c.tprclient.Post().
					Resource("examples").
					Namespace(api.NamespaceDefault).
					Body(example).
					Do().Into(&result)

				if err != nil {
					panic(err)
				}
				fmt.Printf("CREATED: %#v\n", result)
			} else {
				panic(err)
			}
		} else {
			fmt.Printf("GET: %#v\n", example)
		}

	}()

	// p, err := c.client.CoreV1().RESTClient().Get().Resource("thirdpartyresources").Do().Get()
	// p, err := c.client.CoreV1().RESTClient().Get().Resource("thirdpartyresources").Namespace("user001").Do().Get()
	// fmt.Println("###%v,%v", p, err)
}

func podStatus(pod *v1.Pod) string {

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			}
		}
	}
	if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return reason
}

// The current client-go StoreToNodeLister expects api.Node - but client returns v1.Node. Add this shim until next release
type storeToNodeLister struct {
	cache.Store
}

func (s *storeToNodeLister) List() (machines v1.PodList, err error) {
	for _, m := range s.Store.List() {
		machines.Items = append(machines.Items, *(m.(*v1.Pod)))
	}
	return machines, nil
}
