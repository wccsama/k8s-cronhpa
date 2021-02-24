// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	cronhpav1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCronHPAs implements CronHPAInterface
type FakeCronHPAs struct {
	Fake *FakeWccV1
	ns   string
}

var cronhpasResource = schema.GroupVersionResource{Group: "wcc.io", Version: "v1", Resource: "cronhpas"}

var cronhpasKind = schema.GroupVersionKind{Group: "wcc.io", Version: "v1", Kind: "CronHPA"}

// Get takes name of the cronHPA, and returns the corresponding cronHPA object, and an error if there is any.
func (c *FakeCronHPAs) Get(ctx context.Context, name string, options v1.GetOptions) (result *cronhpav1.CronHPA, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(cronhpasResource, c.ns, name), &cronhpav1.CronHPA{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronhpav1.CronHPA), err
}

// List takes label and field selectors, and returns the list of CronHPAs that match those selectors.
func (c *FakeCronHPAs) List(ctx context.Context, opts v1.ListOptions) (result *cronhpav1.CronHPAList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(cronhpasResource, cronhpasKind, c.ns, opts), &cronhpav1.CronHPAList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &cronhpav1.CronHPAList{ListMeta: obj.(*cronhpav1.CronHPAList).ListMeta}
	for _, item := range obj.(*cronhpav1.CronHPAList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cronHPAs.
func (c *FakeCronHPAs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(cronhpasResource, c.ns, opts))

}

// Create takes the representation of a cronHPA and creates it.  Returns the server's representation of the cronHPA, and an error, if there is any.
func (c *FakeCronHPAs) Create(ctx context.Context, cronHPA *cronhpav1.CronHPA, opts v1.CreateOptions) (result *cronhpav1.CronHPA, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(cronhpasResource, c.ns, cronHPA), &cronhpav1.CronHPA{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronhpav1.CronHPA), err
}

// Update takes the representation of a cronHPA and updates it. Returns the server's representation of the cronHPA, and an error, if there is any.
func (c *FakeCronHPAs) Update(ctx context.Context, cronHPA *cronhpav1.CronHPA, opts v1.UpdateOptions) (result *cronhpav1.CronHPA, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(cronhpasResource, c.ns, cronHPA), &cronhpav1.CronHPA{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronhpav1.CronHPA), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeCronHPAs) UpdateStatus(ctx context.Context, cronHPA *cronhpav1.CronHPA, opts v1.UpdateOptions) (*cronhpav1.CronHPA, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(cronhpasResource, "status", c.ns, cronHPA), &cronhpav1.CronHPA{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronhpav1.CronHPA), err
}

// Delete takes name of the cronHPA and deletes it. Returns an error if one occurs.
func (c *FakeCronHPAs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(cronhpasResource, c.ns, name), &cronhpav1.CronHPA{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCronHPAs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(cronhpasResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &cronhpav1.CronHPAList{})
	return err
}

// Patch applies the patch and returns the patched cronHPA.
func (c *FakeCronHPAs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *cronhpav1.CronHPA, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(cronhpasResource, c.ns, name, pt, data, subresources...), &cronhpav1.CronHPA{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronhpav1.CronHPA), err
}
