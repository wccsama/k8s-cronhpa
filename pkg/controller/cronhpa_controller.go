package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	autoscalingclient "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	scaleclient "k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	cronhpav1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
	cronscheme "github.com/k8s-cronhpa/pkg/client/clientset/versioned/scheme"
	clientset "github.com/k8s-cronhpa/pkg/client/clientset/versioned/typed/cronhpa/v1"
	informers "github.com/k8s-cronhpa/pkg/client/informers/externalversions/cronhpa/v1"
	listers "github.com/k8s-cronhpa/pkg/client/listers/cronhpa/v1"
)

type timestampedScaleEvent struct {
	replicaChange int32 // positive for scaleUp, negative for scaleDown
	timestamp     time.Time
	outdated      bool
}

// CronHPAController ..
type CronHPAController struct {
	// kubeClientset operates resource in cluster
	kubeClientset kubernetes.Interface
	// hpaNamespacer operates horizontalpodautoscalers resource
	hpaNamespacer autoscalingclient.HorizontalPodAutoscalersGetter
	// cronHPAClient operates cronHPA resource
	cronHPAClient clientset.CronHPAsGetter
	// scaleNamespacer scales resource
	scaleNamespacer scaleclient.ScalesGetter
	// mapper converts GK to certain resource
	mapper apimeta.RESTMapper
	// eventRecorder records events
	eventRecorder record.EventRecorder
	// cronHPALister lists cronHPA from cache
	cronHPALister listers.CronHPALister
	// cronHPAListerSynced syncs cache in the first time
	cronHPAListerSynced cache.InformerSynced
	// timeNow interface gets now time for testing
	timeNow timeNow
	// controllerResyncPeriod resyncs time
	controllerResyncPeriod time.Duration

	// TODO: add scale events
	// Latest autoscaler events
	scaleUpEvents   map[string][]timestampedScaleEvent
	scaleDownEvents map[string][]timestampedScaleEvent
}

// NewCronHPAController returns CronHPAController
func NewCronHPAController(kubeclientset kubernetes.Interface,
	cronhpaclient clientset.CronHPAsGetter,
	cronHPAInformer informers.CronHPAInformer,
	scaleNamespacer scaleclient.ScalesGetter,
	hpaNamespacer autoscalingclient.HorizontalPodAutoscalersGetter,
	mapper apimeta.RESTMapper,
	controllerResyncPeriod time.Duration) *CronHPAController {

	utilruntime.Must(cronscheme.AddToScheme(scheme.Scheme))
	broadcaster := record.NewBroadcaster()
	broadcaster.StartLogging(klog.Infof)
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	record := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "cron-hpa-controller"})

	cronHPAController := &CronHPAController{
		kubeClientset:          kubeclientset,
		hpaNamespacer:          hpaNamespacer,
		cronHPAClient:          cronhpaclient,
		scaleNamespacer:        scaleNamespacer,
		mapper:                 mapper,
		cronHPALister:          cronHPAInformer.Lister(),
		eventRecorder:          record,
		timeNow:                &timePackage{},
		cronHPAListerSynced:    cronHPAInformer.Informer().HasSynced,
		controllerResyncPeriod: controllerResyncPeriod,
	}

	return cronHPAController
}

// Run ..
func (chc *CronHPAController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	klog.Infof("Starting cron HPA controller")
	defer klog.Infof("Shutting down cron HPA controller")

	if !cache.WaitForCacheSync(stopCh, chc.cronHPAListerSynced) {
		klog.Errorf("failed to sync data")
		return
	}

	go wait.Until(chc.reconcileCronHPAs, chc.controllerResyncPeriod*time.Second, stopCh)

	<-stopCh
}

func (chc *CronHPAController) reconcileCronHPAs() {
	cronHPAs, err := chc.cronHPALister.CronHPAs(metav1.NamespaceAll).List(labels.Everything())
	if err != nil {
		klog.Errorf("List CronHPAs err: %v", err)
		return
	}

	for _, cronHPA := range cronHPAs {
		ctx := context.Background()
		err := chc.reconcileCronHPA(ctx, cronHPA, chc.timeNow.now())
		if err != nil {
			klog.Errorf("reconcileCronHPA: %v, err: %v", getCronHPAFullName(cronHPA), err)
		}
	}
}

func (chc *CronHPAController) reconcileCronHPA(ctx context.Context, cronHPAOrigin *cronhpav1.CronHPA, now time.Time) error {
	cronHPA := cronHPAOrigin.DeepCopy()

	latestSchedledTime := getLatestScheduledTime(cronHPA)
	cronJobBatch := make([]cronhpav1.Cron, 0)
	checkName := make(map[string]int, 0)

	// get exclude date
	excludeTimes, err := getExcludeTimes(latestSchedledTime, now, cronHPA.Spec.ExcludeDates, cronHPA.Spec.StartingDeadlineSeconds)
	if err != nil {
		klog.Errorf("getExcludeTimes err: %v", err)
		return err
	}

	// calculate all the cron strings in spec
	for _, cron := range cronHPA.Spec.Crons {
		// check cron name
		if _, ok := checkName[cron.Name]; ok {
			chc.eventRecorder.Eventf(cronHPA, v1.EventTypeWarning, fmt.Sprintf("cronName duplication: %v", cron.Name), "")
			klog.Errorf("cronName duplication name: %v", cron.Name)
			return fmt.Errorf("cronName duplication name: %v", cron.Name)
		} else {
			checkName[cron.Name] = 1
		}

		// job has been run
		if cron.RunOnce && haveRunAlready(cron, cronHPA) {
			continue
		}

		// get job's time
		times, err := getRecentUnmetScheduleTimes(latestSchedledTime, now, cron.Schedule, cronHPA.Spec.StartingDeadlineSeconds)
		if err != nil {
			klog.Errorf("getRecentUnmetScheduleTimes err: %v", err)
			return err
		}

		// cron is not ready for running
		if len(times) == 0 {
			continue
		}

		// filter with exclude time
		if shoudFilterTimesWithExcludeTimes(excludeTimes, times[len(times)-1]) {
			klog.Infof("[ExcludeDates] try to excludeDates: %v, latestSchedledTime: %v, schedule: %v, scheduleTime: %v",
				excludeTimes, latestSchedledTime, cron.Schedule, times[len(times)-1])
			continue
		}

		// log some helpful message
		klog.Infof("[Schedule] for %s of cronhpa %v latestSchedledTime: %v", cron.Schedule, times[len(times)-1], latestSchedledTime)
		klog.Infof("[Plan] %v can be scaled to replicas %v for schedule %v", getCronHPAFullName(cronHPA),
			cron.TargeSize, cron.Schedule)
		// add task to batch
		cronJobBatch = append(cronJobBatch, cron)
	}

	if len(cronJobBatch) > 0 {
		// sort batch, big to small
		sort.Slice(cronJobBatch, func(i, j int) bool { return cronJobBatch[i].TargeSize > cronJobBatch[j].TargeSize })
		klog.Infof("[Scaled] %v can be scale to replicas %v", getCronHPAFullName(cronHPA),
			cronJobBatch[0].TargeSize)

		// multiple job choose the highest replicas one
		if err := chc.updateScaleTargetRef(ctx, cronHPA, cronJobBatch[0].TargeSize); err != nil {
			klog.Errorf("updateScaleTargetRef Failed to scale %v to replicas %v: %v", getCronHPAFullName(cronHPA), cronJobBatch[0], err)
			return err
		}

		// Update status when scale successfully
		cronHPA.Status.LastScheduleTime = &metav1.Time{Time: chc.timeNow.now()}
		cronHPA.Status.Conditions = updateConditions(
			cronHPA.Status.Conditions,
			cronhpav1.CronHPAConditions{
				Name:    cronJobBatch[0].Name,
				Status:  cronhpav1.Succeed,
				RunOnce: cronJobBatch[0].RunOnce})

		if _, err := chc.cronHPAClient.CronHPAs(cronHPA.Namespace).Update(context.TODO(), cronHPA, metav1.UpdateOptions{}); err != nil {
			chc.eventRecorder.Eventf(cronHPA, v1.EventTypeWarning, fmt.Sprintf("FailedUpdateStatus cronHPA: %v", getCronHPAFullName(cronHPA)), err.Error())
			klog.Errorf("Failed to update cronhpa %v's LastScheduleTime(%+v): %v",
				getCronHPAFullName(cronHPA), cronHPA.Status.LastScheduleTime.Time, err)
			return err
		}
	}

	return nil
}

func (chc *CronHPAController) updateScaleTargetRef(ctx context.Context, cronHPAV1 *cronhpav1.CronHPA, targeSize int32) (err error) {
	if cronHPAV1.Spec.ScaleTargetRef.Kind == "HorizontalPodAutoscaler" {
		err = chc.updateHPAIfNeed(ctx, cronHPAV1, targeSize)
	} else {
		err = chc.tryToScale(ctx, cronHPAV1, cronHPAV1.Spec.ScaleTargetRef, targeSize)
	}

	return err
}

// updateHPAIfNeed
func (chc *CronHPAController) updateHPAIfNeed(ctx context.Context, cronHPAV1 *cronhpav1.CronHPA, targeSize int32) error {
	var needUpdate bool
	var NewTargeSize int32

	// try to get hpa
	hpa, err := chc.hpaNamespacer.HorizontalPodAutoscalers(cronHPAV1.Namespace).Get(context.TODO(), cronHPAV1.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("get hpa: %v/%v err: %v", cronHPAV1.Namespace, cronHPAV1.Spec.ScaleTargetRef.Name, err)
		return err
	}

	if targeSize < *hpa.Spec.MinReplicas {
		*hpa.Spec.MinReplicas = targeSize
		needUpdate = true
	}

	if targeSize > hpa.Spec.MaxReplicas {
		hpa.Spec.MaxReplicas = targeSize
		needUpdate = true
	}

	if targeSize > hpa.Status.CurrentReplicas {
		*hpa.Spec.MinReplicas = targeSize
		NewTargeSize = targeSize
		needUpdate = true
	}

	if needUpdate {
		_, err := chc.hpaNamespacer.HorizontalPodAutoscalers(cronHPAV1.Namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Update HPA min/max err. HPA: %v err: %v", hpa, err)
			chc.eventRecorder.Eventf(cronHPAV1, v1.EventTypeWarning, fmt.Sprintf("FailedUpdateStatus HPA: %v/%v", hpa.Namespace, hpa.Name), err.Error())
			return err
		}
		chc.eventRecorder.Eventf(cronHPAV1, v1.EventTypeNormal, "SuccessfulUpdateHPA", fmt.Sprintf("New min/max: %v/%v", *hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas))
	}

	// try to keep the highest replicas
	if NewTargeSize > hpa.Status.CurrentReplicas {
		scaleTargetRef := cronhpav1.ScaleTargetRef{
			Name:       hpa.Spec.ScaleTargetRef.Name,
			Kind:       hpa.Spec.ScaleTargetRef.Kind,
			APIVersion: hpa.Spec.ScaleTargetRef.APIVersion,
		}

		err = chc.tryToScale(ctx, cronHPAV1, scaleTargetRef, NewTargeSize)
		if err != nil {
			klog.Errorf("tryToScale resource %v in hpa err: %v", scaleTargetRef, err)

			return err
		}
	}

	return nil
}

// tryToScale try to scale scaleTargetRef in the func's parameter. cronHPAV1 is just for namespace and logging
func (chc *CronHPAController) tryToScale(ctx context.Context, cronHPAV1 *cronhpav1.CronHPA, scaleTargetRef cronhpav1.ScaleTargetRef, targeSize int32) error {
	// TODO: checkout targetSize
	reference := fmt.Sprintf("%s/%s/%s", scaleTargetRef.Kind, cronHPAV1.Namespace, scaleTargetRef.Name)
	targetGV, err := schema.ParseGroupVersion(scaleTargetRef.APIVersion)
	if err != nil {
		klog.Errorf("failed to parseGroupVersion err: %v, cronHPAV1: %v", err, getCronHPAFullName(cronHPAV1))
		return err
	}

	targetGK := schema.GroupKind{
		Group: targetGV.Group,
		Kind:  scaleTargetRef.Kind,
	}

	mappings, err := chc.mapper.RESTMappings(targetGK)
	if err != nil {
		klog.Errorf("failed to RESTMappings for targetGK: %v err: %v, cronHPAV1: %v", targetGK, err, getCronHPAFullName(cronHPAV1))
		return err
	}

	scale, targetGR, err := chc.scaleForResourceMappings(cronHPAV1.Namespace, scaleTargetRef.Name, mappings)
	if err != nil {
		klog.Errorf("failed to scaleForResourceMappings err: %v, cronHPAV1: %v", err, getCronHPAFullName(cronHPAV1))
		return err
	}

	if scale.Spec.Replicas != targeSize {
		oldReplicas := scale.Spec.Replicas
		scale.Spec.Replicas = targeSize
		_, err = chc.scaleNamespacer.Scales(cronHPAV1.Namespace).Update(context.TODO(), targetGR, scale, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to scales err: %v, cronHPAV1: %v", err, getCronHPAFullName(cronHPAV1))
			chc.eventRecorder.Eventf(cronHPAV1, v1.EventTypeWarning, fmt.Sprintf("failed to scale %v cronHPAV1: %v", reference, cronHPAV1), err.Error())
			return fmt.Errorf("failed to rescale %s: %v", reference, err)
		}
		chc.eventRecorder.Eventf(cronHPAV1, v1.EventTypeNormal, "SuccessfulRescale", "New size: %d", targeSize)
		klog.Infof("Successful scale of %s, old size: %d, new size: %d",
			getCronHPAFullName(cronHPAV1), oldReplicas, targeSize)
	} else {
		klog.Infof("No need to scale %s to %v, same replicas", getCronHPAFullName(cronHPAV1), targeSize)
	}

	return nil
}

func (chc *CronHPAController) scaleForResourceMappings(namespace, name string, mappings []*apimeta.RESTMapping) (*autoscalingv1.Scale, schema.GroupResource, error) {
	var firstErr error
	for i, mapping := range mappings {
		targetGR := mapping.Resource.GroupResource()
		scale, err := chc.scaleNamespacer.Scales(namespace).Get(context.TODO(), targetGR, name, metav1.GetOptions{})
		if err == nil {
			return scale, targetGR, nil
		}

		// if this is the first error, remember it,
		// then go on and try other mappings until we find a good one
		if i == 0 {
			firstErr = err
		}
	}

	// make sure we handle an empty set of mappings
	if firstErr == nil {
		firstErr = fmt.Errorf("unrecognized resource")
	}

	return nil, schema.GroupResource{}, firstErr
}

// GetEventRecorder for unit test
func (chc *CronHPAController) GetEventRecorder() record.EventRecorder {
	return chc.eventRecorder
}
