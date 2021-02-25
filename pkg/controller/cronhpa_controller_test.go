package controller

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	sacleFakeClient "k8s.io/client-go/scale/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	cronHPAV1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
	cronHPAFakeClient "github.com/k8s-cronhpa/pkg/client/clientset/versioned/fake"
	informersF "github.com/k8s-cronhpa/pkg/client/informers/externalversions"
	informers "github.com/k8s-cronhpa/pkg/client/informers/externalversions/internalinterfaces"
)

type fakeResource struct {
	name       string
	apiVersion string
	kind       string
}

type fakeTimeNow struct {
	fakeTime time.Time
}

func (ftn *fakeTimeNow) now() time.Time {
	return ftn.fakeTime
}

type testCase struct {
	sync.Mutex
	expectedDesiredReplicas int32
	initialReplicas         int32
	fakeTimeNow             time.Time
	crons                   []cronHPAV1.Cron

	// Channel with names of HPA objects which we have reconciled.
	processed     chan string
	scaleUpdated  bool
	statusUpdated bool

	// Target resource information.
	resource             *fakeResource
	hpaReferenceResource *fakeResource

	// HPA test spec
	hpaMin     *int32
	hpaMax     int32
	hpaCurrent int32

	// expect hpa spec
	expectHPAMin *int32
	expectHPAMax int32

	// Last scale time
	lastScaleTime time.Time
	createTime    time.Time
}

func alwaysReady() bool { return true }

func (tc *testCase) prepareClient(t *testing.T) (*fake.Clientset, *cronHPAFakeClient.Clientset, *sacleFakeClient.FakeScaleClient) {
	namespace := "test-namespace"
	cronHPAName := "test-cronhpa"

	tc.Lock()
	tc.processed = make(chan string, 100)
	if tc.resource == nil {
		tc.resource = &fakeResource{
			name:       "test-rc",
			apiVersion: "v1",
			kind:       "ReplicationController",
		}
	}
	tc.Unlock()

	kubeClientset := &fake.Clientset{}
	kubeClientset.AddReactor("get", "horizontalpodautoscalers", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestHPA",
				Namespace: namespace,
			},
			Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
				MinReplicas: tc.hpaMin,
				MaxReplicas: tc.hpaMax,
				ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
					Kind:       tc.hpaReferenceResource.kind,
					APIVersion: tc.hpaReferenceResource.apiVersion,
					Name:       tc.hpaReferenceResource.name,
				},
			},
			Status: autoscalingv1.HorizontalPodAutoscalerStatus{
				CurrentReplicas: tc.hpaCurrent,
			},
		}

		return true, obj, nil
	})

	kubeClientset.AddReactor("update", "horizontalpodautoscalers", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := action.(core.UpdateAction).GetObject().(*autoscalingv1.HorizontalPodAutoscaler)
		assert.Equal(t, *tc.expectHPAMin, *obj.Spec.MinReplicas, "the HPA expectHPAMin should be as expected")
		assert.Equal(t, tc.expectHPAMax, obj.Spec.MaxReplicas, "the HPA MaxReplicas should be as expected")
		// assert.Equal(t, tc.expectedDesiredReplicas, obj.Status.DesiredReplicas, "the desired replica count reported in the object status should be as expected")

		return true, obj, nil
	})

	cronHPAClientset := &cronHPAFakeClient.Clientset{}
	cronHPAClientset.AddReactor("list", "cronhpas", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := &cronHPAV1.CronHPAList{
			Items: []cronHPAV1.CronHPA{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              cronHPAName,
						Namespace:         namespace,
						CreationTimestamp: metav1.Time{Time: tc.createTime},
						SelfLink:          "experimental/v1/namespaces/" + namespace + "/cronhpas/" + cronHPAName,
					},
					Spec: cronHPAV1.CronHPASpec{
						ScaleTargetRef: cronHPAV1.ScaleTargetRef{
							Kind:       tc.resource.kind,
							Name:       tc.resource.name,
							APIVersion: tc.resource.apiVersion,
						},
						Crons: tc.crons,
					},
					Status: cronHPAV1.CronHPAStatus{
						LastScheduleTime: &metav1.Time{Time: tc.lastScaleTime},
					},
				},
			},
		}

		return true, obj, nil
	})

	cronHPAClientset.AddReactor("get", "cronhpas", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := &cronHPAV1.CronHPA{
			ObjectMeta: metav1.ObjectMeta{
				Name:              cronHPAName,
				Namespace:         namespace,
				CreationTimestamp: metav1.Time{Time: tc.createTime},
				SelfLink:          "experimental/v1/namespaces/" + namespace + "/cronhpas/" + cronHPAName,
			},
			Spec: cronHPAV1.CronHPASpec{
				ScaleTargetRef: cronHPAV1.ScaleTargetRef{
					Kind:       tc.resource.kind,
					Name:       tc.resource.name,
					APIVersion: tc.resource.apiVersion,
				},
				Crons: tc.crons,
			},
			Status: cronHPAV1.CronHPAStatus{
				LastScheduleTime: &metav1.Time{Time: tc.lastScaleTime},
			},
		}

		return true, obj, nil
	})

	cronHPAClientset.AddReactor("update", "cronhpas", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := action.(core.UpdateAction).GetObject().(*cronHPAV1.CronHPA)
		assert.Equal(t, namespace, obj.Namespace, "the cronHPA namespace should be as expected")
		assert.Equal(t, cronHPAName, obj.Name, "the cronHPA name should be as expected")
		// assert.Equal(t, tc.expectedDesiredReplicas, obj.Status.DesiredReplicas, "the desired replica count reported in the object status should be as expected")

		tc.statusUpdated = true
		// Every time we reconcile HPA object we are updating status.
		tc.processed <- obj.Name
		return true, obj, nil
	})

	fakeScaleClient := &sacleFakeClient.FakeScaleClient{}
	fakeScaleClient.AddReactor("get", "replicationcontrollers", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tc.resource.name,
				Namespace: namespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: tc.initialReplicas,
			},
			Status: autoscalingv1.ScaleStatus{
				Replicas: tc.initialReplicas,
			},
		}
		return true, obj, nil
	})

	fakeScaleClient.AddReactor("get", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tc.resource.name,
				Namespace: namespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: tc.initialReplicas,
			},
			Status: autoscalingv1.ScaleStatus{
				Replicas: tc.initialReplicas,
			},
		}
		return true, obj, nil
	})

	fakeScaleClient.AddReactor("get", "replicasets", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tc.resource.name,
				Namespace: namespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: tc.initialReplicas,
			},
			Status: autoscalingv1.ScaleStatus{
				Replicas: tc.initialReplicas,
			},
		}
		return true, obj, nil
	})

	fakeScaleClient.AddReactor("update", "replicationcontrollers", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := action.(core.UpdateAction).GetObject().(*autoscalingv1.Scale)
		replicas := action.(core.UpdateAction).GetObject().(*autoscalingv1.Scale).Spec.Replicas
		assert.Equal(t, tc.expectedDesiredReplicas, replicas, "the replica count of the RC should be as expected")
		tc.scaleUpdated = true
		return true, obj, nil
	})

	fakeScaleClient.AddReactor("update", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := action.(core.UpdateAction).GetObject().(*autoscalingv1.Scale)
		replicas := action.(core.UpdateAction).GetObject().(*autoscalingv1.Scale).Spec.Replicas
		assert.Equal(t, tc.expectedDesiredReplicas, replicas, "the replica count of the deployment should be as expected")
		tc.scaleUpdated = true
		return true, obj, nil
	})

	fakeScaleClient.AddReactor("update", "replicasets", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		tc.Lock()
		defer tc.Unlock()

		obj := action.(core.UpdateAction).GetObject().(*autoscalingv1.Scale)
		replicas := action.(core.UpdateAction).GetObject().(*autoscalingv1.Scale).Spec.Replicas
		assert.Equal(t, tc.expectedDesiredReplicas, replicas, "the replica count of the replicaset should be as expected")
		tc.scaleUpdated = true
		return true, obj, nil
	})

	return kubeClientset, cronHPAClientset, fakeScaleClient
}

func (tc *testCase) setUpController(t *testing.T) (*CronHPAController, informers.SharedInformerFactory) {
	kubeClientset, cronHPAClientset, scaleNamespacer := tc.prepareClient(t)

	mapper := testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...)
	informerFactory := informersF.NewSharedInformerFactory(cronHPAClientset, 0)

	controller := NewCronHPAController(
		kubeClientset,
		cronHPAClientset.WccV1(),
		informerFactory.Wcc().V1().CronHPAs(),
		scaleNamespacer,
		kubeClientset.AutoscalingV1(),
		mapper,
		time.Duration(0))

	controller.cronHPAListerSynced = alwaysReady
	controller.eventRecorder = record.NewFakeRecorder(100)
	controller.timeNow = &fakeTimeNow{tc.fakeTimeNow}

	return controller, informerFactory
}

func (tc *testCase) runTestWithController(t *testing.T, cronHPAController *CronHPAController, informers informers.SharedInformerFactory) {
	stop := make(chan struct{})
	defer close(stop)

	informers.Start(stop)
	go cronHPAController.Run(stop)

	<-tc.processed
	tc.verifyResults(t)
}

func (tc *testCase) runTest(t *testing.T) {
	cronHPAController, informerFactory := tc.setUpController(t)
	tc.runTestWithController(t, cronHPAController, informerFactory)
}

func (tc *testCase) verifyResults(t *testing.T) {
	tc.Lock()
	defer tc.Unlock()

	assert.Equal(t, tc.initialReplicas != tc.expectedDesiredReplicas, tc.scaleUpdated, "the scale should only be updated if we expected a change in replicas")
	//assert.True(t, tc.statusUpdated, "the status should have been updated")
}

func TestScaleUp(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	tc := testCase{
		expectedDesiredReplicas: 3,
		initialReplicas:         2,
		fakeTimeNow:             fakeTimeNow,
		createTime:              createTime,
		lastScaleTime:           lastScaleTime,
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-scale-up",
				Schedule:  "0 * * * ?",
				TargeSize: 3,
			},
		},
	}
	tc.runTest(t)
}

func TestScaleDown(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	tc := testCase{
		expectedDesiredReplicas: 2,
		initialReplicas:         3,
		fakeTimeNow:             fakeTimeNow,
		createTime:              createTime,
		lastScaleTime:           lastScaleTime,
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-scale-up",
				Schedule:  "0 * * * ?",
				TargeSize: 2,
			},
		},
	}
	tc.runTest(t)
}

func TestNotScale(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	tc := testCase{
		expectedDesiredReplicas: 2,
		initialReplicas:         2,
		fakeTimeNow:             fakeTimeNow,
		createTime:              createTime,
		lastScaleTime:           lastScaleTime,
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-scale-up",
				Schedule:  "0 * * * ?",
				TargeSize: 2,
			},
		},
	}
	tc.runTest(t)
}

// cronHPA and hpa case 1
// HPA(MIN/MAX) | cronHPA(targetSize) | deploy(currentSize) | result
// 1/10         | 5                   | 5                   | hpa(1/10) deploy 5
// notice: initialReplicas is equal to hpaCurrent
func TestHPAEqualCronHPA(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	var num int32 = 1

	tc := testCase{
		expectedDesiredReplicas: 5,
		initialReplicas:         5,
		hpaMin:                  &num,
		hpaMax:                  10,
		hpaCurrent:              5,

		expectHPAMin:  &num,
		expectHPAMax:  10,
		fakeTimeNow:   fakeTimeNow,
		createTime:    createTime,
		lastScaleTime: lastScaleTime,
		hpaReferenceResource: &fakeResource{
			name:       "test-rc",
			apiVersion: "v1",
			kind:       "ReplicationController",
		},
		resource: &fakeResource{
			name:       "test-hpa",
			apiVersion: "v1",
			kind:       "HorizontalPodAutoscaler",
		},
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-don't-scale",
				Schedule:  "0 * * * ?",
				TargeSize: 5,
			},
		},
	}
	tc.runTest(t)
}

// cronHPA and hpa case 2
// HPA(MIN/MAX) | cronHPA(targetSize) | deploy(currentSize) | result
// 1/10         | 4                   | 5                   | hpa(1/10) deploy 5
// notice: initialReplicas is equal to hpaCurrent
func TestHPAMoreThanCronHPA(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	var num int32 = 1

	tc := testCase{
		expectedDesiredReplicas: 5,
		initialReplicas:         5,
		hpaMin:                  &num,
		hpaMax:                  10,
		hpaCurrent:              5,

		expectHPAMin:  &num,
		expectHPAMax:  10,
		fakeTimeNow:   fakeTimeNow,
		createTime:    createTime,
		lastScaleTime: lastScaleTime,
		hpaReferenceResource: &fakeResource{
			name:       "test-rc",
			apiVersion: "v1",
			kind:       "ReplicationController",
		},
		resource: &fakeResource{
			name:       "test-hpa",
			apiVersion: "v1",
			kind:       "HorizontalPodAutoscaler",
		},
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-don't-scale",
				Schedule:  "0 * * * ?",
				TargeSize: 4,
			},
		},
	}
	tc.runTest(t)
}

// cronHPA and hpa case 3
// HPA(MIN/MAX) | cronHPA(targetSize) | deploy(currentSize) | result
// 1/10         | 6                   | 5                   | hpa(6/10) deploy 6
// notice: initialReplicas is equal to hpaCurrent
func TestHPALessThanCronHPA(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	var min int32 = 1
	var expect int32 = 6

	tc := testCase{
		expectedDesiredReplicas: 6,
		initialReplicas:         5,
		hpaMin:                  &min,
		hpaMax:                  10,
		hpaCurrent:              5,

		expectHPAMin:  &expect,
		expectHPAMax:  10,
		fakeTimeNow:   fakeTimeNow,
		createTime:    createTime,
		lastScaleTime: lastScaleTime,
		hpaReferenceResource: &fakeResource{
			name:       "test-rc",
			apiVersion: "v1",
			kind:       "ReplicationController",
		},
		resource: &fakeResource{
			name:       "test-hpa",
			apiVersion: "v1",
			kind:       "HorizontalPodAutoscaler",
		},
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-scale-up",
				Schedule:  "0 * * * ?",
				TargeSize: 6,
			},
		},
	}
	tc.runTest(t)
}

// cronHPA and hpa case 4
// HPA(MIN/MAX) | cronHPA(targetSize) | deploy(currentSize) | result
// 5/10         | 4                   | 5                   | hpa(4/10) deploy 5
// notice: initialReplicas is equal to hpaCurrent
func TestCronHPALessThanMin(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	var min int32 = 5
	var expect int32 = 4

	tc := testCase{
		expectedDesiredReplicas: 5,
		initialReplicas:         5,
		hpaMin:                  &min,
		hpaMax:                  10,
		hpaCurrent:              5,

		expectHPAMin:  &expect,
		expectHPAMax:  10,
		fakeTimeNow:   fakeTimeNow,
		createTime:    createTime,
		lastScaleTime: lastScaleTime,
		hpaReferenceResource: &fakeResource{
			name:       "test-rc",
			apiVersion: "v1",
			kind:       "ReplicationController",
		},
		resource: &fakeResource{
			name:       "test-hpa",
			apiVersion: "v1",
			kind:       "HorizontalPodAutoscaler",
		},
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-don't-scale",
				Schedule:  "0 * * * ?",
				TargeSize: 5,
			},
		},
	}
	tc.runTest(t)
}

// cronHPA and hpa case 5
// HPA(MIN/MAX) | cronHPA(targetSize) | deploy(currentSize) | result
// 5/10         | 11                  | 5                   | hpa(11/11) deploy 11
// notice: initialReplicas is equal to hpaCurrent
func TestCronHPAMoreThanMax(t *testing.T) {
	fakeTimeNow, err := time.Parse(time.RFC3339, "2020-05-19T14:01:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	createTime, err := time.Parse(time.RFC3339, "2020-05-19T13:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	lastScaleTime, err := time.Parse(time.RFC3339, "2020-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("unexpect parse time err: %v", err)
	}

	var min int32 = 5
	var expect int32 = 11

	tc := testCase{
		expectedDesiredReplicas: 11,
		initialReplicas:         5,
		hpaMin:                  &min,
		hpaMax:                  10,
		hpaCurrent:              5,

		expectHPAMin:  &expect,
		expectHPAMax:  expect,
		fakeTimeNow:   fakeTimeNow,
		createTime:    createTime,
		lastScaleTime: lastScaleTime,
		hpaReferenceResource: &fakeResource{
			name:       "test-rc",
			apiVersion: "v1",
			kind:       "ReplicationController",
		},
		resource: &fakeResource{
			name:       "test-hpa",
			apiVersion: "v1",
			kind:       "HorizontalPodAutoscaler",
		},
		crons: []cronHPAV1.Cron{
			cronHPAV1.Cron{
				Name:      "test-scale-up",
				Schedule:  "0 * * * ?",
				TargeSize: 11,
			},
		},
	}
	tc.runTest(t)
}

// add more tests
