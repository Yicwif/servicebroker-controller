package main

import (
	"flag"
	"fmt"

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

	glog.Infof("Starting ServiceBroker controller, version: %v", Version)
	go serviceBrokerController(client, tprclient).controller.Run(wait.NeverStop)
	glog.Infof("Starting ServiceInstance controller, version: %v", Version)
	go serviceInstanceController(client, tprclient).controller.Run(wait.NeverStop)
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
						{Name: TPR_VERSION},
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
		Group:   TPR_GROUP,
		Version: TPR_VERSION,
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
				&ServiceInstance{},
				&ServiceInstanceList{},
				&api.ListOptions{},
				&api.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(api.Scheme)
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
