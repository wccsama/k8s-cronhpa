package controller

import (
	"context"
	"reflect"

	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var CRD = &extensionsobj.CustomResourceDefinition{
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
			Plural:   "cronhpas",
			Singular: "cronhpa",
			Kind:     "CronHPA",
			ListKind: "CronHPAList",
		},
	},
}

// EnsureCRDCreated ensures crd being created
func EnsureCRDCreated(client apiextensionsclient.Interface) (created bool, err error) {
	crdClient := client.ApiextensionsV1beta1().CustomResourceDefinitions()
	presetCRD, err := crdClient.Get(context.TODO(), CRD.Name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			if _, err := crdClient.Create(context.TODO(), CRD, metav1.CreateOptions{}); err != nil {
				klog.Errorf("Error creating CRD %s: %v", CRD.Name, err)
				return false, err
			}
			klog.V(1).Infof("Create CRD %s successfully.", CRD.Name)
			return true, nil
		}
		klog.Errorf("Error getting CRD %s: %v", CRD.Name, err)
		return false, err
	}

	if reflect.DeepEqual(presetCRD.Spec, CRD.Spec) {
		klog.V(1).Infof("CRD %s already exists", CRD.Name)
	} else {
		klog.V(3).Infof("Update CRD %s: %+v -> %+v", CRD.Name, presetCRD.Spec, CRD.Spec)
		newCRD := CRD
		newCRD.ResourceVersion = presetCRD.ResourceVersion
		// Update CRD
		if _, err := crdClient.Update(context.TODO(), newCRD, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Error update CRD %s: %v", CRD.Name, err)
			return false, err
		}
		klog.V(1).Infof("Update CRD %s successfully.", CRD.Name)
	}

	return true, nil
}
