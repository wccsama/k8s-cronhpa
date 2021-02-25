package controller

import (
	"context"
	"testing"

	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var fakeCRD = &extensionsobj.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "cronhpas.wcc.io",
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "CustomResourceDefinition",
		APIVersion: "apiextensions.k8s.io/v1beta1",
	},
	Spec: extensionsobj.CustomResourceDefinitionSpec{
		Group:   "wcc.io",
		Version: "v1",
		Scope:   extensionsobj.ResourceScope("Namespaced"),
		Names: extensionsobj.CustomResourceDefinitionNames{
			Plural:   "cronhpa",
			Singular: "cronhp",
			Kind:     "CronHP",
			ListKind: "CronHPALis",
		},
	},
}

func TestEnsureCRDCreated(t *testing.T) {
	testCases := []struct {
		execFunc  func(fakeClient *fake.Clientset)
		expectErr bool
		created   bool
	}{
		{
			// Case1: get crd nil and create
			execFunc: func(fakeClient *fake.Clientset) {
				return
			},
			expectErr: false,
			created:   true,
		},
		{
			// Case2: get crd already exist
			execFunc: func(fakeClient *fake.Clientset) {
				fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.TODO(), CRD, metav1.CreateOptions{})
			},
			expectErr: false,
			created:   true,
		},
		{
			// Case3: get crd already exist, need be updated
			execFunc: func(fakeClient *fake.Clientset) {
				fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.TODO(), fakeCRD, metav1.CreateOptions{})
			},
			expectErr: false,
			created:   true,
		},
	}

	for _, testCase := range testCases {
		fakeClient := fake.NewSimpleClientset()
		testCase.execFunc(fakeClient)
		resultCeated, err := EnsureCRDCreated(fakeClient)
		if err != nil && !testCase.expectErr {
			t.Errorf("unexpect err: %v", err)
		}
		if resultCeated != testCase.created {
			t.Errorf("EnsureCRDCreated expect created: %v, get: %v", testCase.created, resultCeated)

		}
	}
}
