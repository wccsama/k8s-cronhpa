package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/util/wait"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	scaleclient "k8s.io/client-go/scale"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog"

	clientset "github.com/k8s-cronhpa/pkg/client/clientset/versioned"
	informers "github.com/k8s-cronhpa/pkg/client/informers/externalversions"
	controller "github.com/k8s-cronhpa/pkg/controller"
)

const (
	defaultLeaseDuration = 15 * time.Second
	defaultRenewDeadline = 10 * time.Second
	defaultRetryPeriod   = 2 * time.Second
)

var (
	createCRD              bool
	controllerResyncPeriod int32
	informerDefaultResync  int32
	leaderElect            bool
)

func addFlags(fs *pflag.FlagSet) {
	// TODO: support out of cluster
	fs.Int32Var(&controllerResyncPeriod, "controllerResyncPeriod", 60, "controller resyncPeriod unit second")
	fs.Int32Var(&informerDefaultResync, "informerDefaultResync", 30, "informer resyncPeriod unit second")
	fs.BoolVar(&createCRD, "create-crd", true, "Create cronhpa CRD if it does not exist")
	fs.BoolVar(&leaderElect, "leaderElect", false, "open leaderElection or not")
}

func main() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	addFlags(pflag.CommandLine)
	pflag.Parse()

	if err := checkCommandParameters(); err != nil {
		klog.Fatalf("checkCommandParameters err: %s", err.Error())
	}

	// TODO: multiple copy
	clientConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Error getting kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	cronHPAClient, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatalf("Error building cronHpa clientset: %s", err.Error())
	}

	// For ensuring crd
	extensionsClient, err := apiextensionsclient.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatalf("Error instantiating apiextensions client: %s", err.Error())
	}

	// For creating controller
	rootClientBuilder := controller.SimpleControllerClientBuilder{
		ClientConfig: clientConfig,
	}

	// Use a discovery client capable of being refreshed.
	discoveryClient := rootClientBuilder.ClientOrDie("cron-hpa-controller")
	cronhpaClientConfig := rootClientBuilder.ConfigOrDie("cron-hpa-controller")
	cachedClient := cacheddiscovery.NewMemCacheClient(discoveryClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)

	// we don't use cached discovery because DiscoveryScaleKindResolver does its own caching,
	// so we want to re-fetch every time when we actually ask for it
	scaleKindResolver := scaleclient.NewDiscoveryScaleKindResolver(discoveryClient.Discovery())
	scaleClient, err := scaleclient.NewForConfig(cronhpaClientConfig, restMapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)
	if err != nil {
		klog.Fatalf("scaleclient NewForConfig err: %v", err)
	}

	informerCronHPAFactory := informers.NewSharedInformerFactory(cronHPAClient, time.Duration(informerDefaultResync)*time.Second)
	cronHPAController := controller.NewCronHPAController(kubeClient,
		cronHPAClient.WccV1(),
		informerCronHPAFactory.Wcc().V1().CronHPAs(),
		scaleClient,
		kubeClient.AutoscalingV1(),
		restMapper,
		time.Duration(controllerResyncPeriod))

	run := func() {
		if createCRD {
			wait.PollUntil(time.Second*5, func() (bool, error) { return controller.EnsureCRDCreated(extensionsClient) }, context.Background().Done())
		}

		stopCh := make(chan struct{})
		go wait.Until(func() { restMapper.Reset() }, 30*time.Second, stopCh)
		go informerCronHPAFactory.Start(stopCh)
		cronHPAController.Run(stopCh)
	}

	if !leaderElect {
		run()
	} else {
		id, err := os.Hostname()
		if err != nil {
			klog.Fatalf("Unable to get hostname: %v", err)
		}

		leaderElectionClient := kubernetes.NewForConfigOrDie(rest.AddUserAgent(clientConfig, "cron-hpa-leader-election"))
		rl, err := resourcelock.New(resourcelock.EndpointsResourceLock,
			"kube-system",
			"k8s-cronhpa-controller",
			leaderElectionClient.CoreV1(),
			leaderElectionClient.CoordinationV1(),
			resourcelock.ResourceLockConfig{
				Identity:      id,
				EventRecorder: cronHPAController.GetEventRecorder(),
			})
		if err != nil {
			klog.Fatalf("error creating lock: %v", err)
		}

		leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: defaultLeaseDuration,
			RenewDeadline: defaultRenewDeadline,
			RetryPeriod:   defaultRetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(_ context.Context) {
					run()
				},
				OnStoppedLeading: func() {
					klog.Fatalf("leaderelection lost")
				},
			},
		})

	}
}

func checkCommandParameters() error {
	if controllerResyncPeriod < 0 || controllerResyncPeriod > 60 {
		return fmt.Errorf("controllerResyncPeriod is out of range 0~60. get: %v", controllerResyncPeriod)
	}

	if informerDefaultResync < 0 || informerDefaultResync > 60 {
		return fmt.Errorf("controllerResyncPeriod is out of range 0~60. get: %v", controllerResyncPeriod)
	}

	return nil
}
