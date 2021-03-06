//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	backupv1 "github.com/arangodb/kube-arangodb/pkg/apis/backup/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeArangoBackupPolicies implements ArangoBackupPolicyInterface
type FakeArangoBackupPolicies struct {
	Fake *FakeBackupV1
	ns   string
}

var arangobackuppoliciesResource = schema.GroupVersionResource{Group: "backup.arangodb.com", Version: "v1", Resource: "arangobackuppolicies"}

var arangobackuppoliciesKind = schema.GroupVersionKind{Group: "backup.arangodb.com", Version: "v1", Kind: "ArangoBackupPolicy"}

// Get takes name of the arangoBackupPolicy, and returns the corresponding arangoBackupPolicy object, and an error if there is any.
func (c *FakeArangoBackupPolicies) Get(name string, options v1.GetOptions) (result *backupv1.ArangoBackupPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(arangobackuppoliciesResource, c.ns, name), &backupv1.ArangoBackupPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*backupv1.ArangoBackupPolicy), err
}

// List takes label and field selectors, and returns the list of ArangoBackupPolicies that match those selectors.
func (c *FakeArangoBackupPolicies) List(opts v1.ListOptions) (result *backupv1.ArangoBackupPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(arangobackuppoliciesResource, arangobackuppoliciesKind, c.ns, opts), &backupv1.ArangoBackupPolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &backupv1.ArangoBackupPolicyList{ListMeta: obj.(*backupv1.ArangoBackupPolicyList).ListMeta}
	for _, item := range obj.(*backupv1.ArangoBackupPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested arangoBackupPolicies.
func (c *FakeArangoBackupPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(arangobackuppoliciesResource, c.ns, opts))

}

// Create takes the representation of a arangoBackupPolicy and creates it.  Returns the server's representation of the arangoBackupPolicy, and an error, if there is any.
func (c *FakeArangoBackupPolicies) Create(arangoBackupPolicy *backupv1.ArangoBackupPolicy) (result *backupv1.ArangoBackupPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(arangobackuppoliciesResource, c.ns, arangoBackupPolicy), &backupv1.ArangoBackupPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*backupv1.ArangoBackupPolicy), err
}

// Update takes the representation of a arangoBackupPolicy and updates it. Returns the server's representation of the arangoBackupPolicy, and an error, if there is any.
func (c *FakeArangoBackupPolicies) Update(arangoBackupPolicy *backupv1.ArangoBackupPolicy) (result *backupv1.ArangoBackupPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(arangobackuppoliciesResource, c.ns, arangoBackupPolicy), &backupv1.ArangoBackupPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*backupv1.ArangoBackupPolicy), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeArangoBackupPolicies) UpdateStatus(arangoBackupPolicy *backupv1.ArangoBackupPolicy) (*backupv1.ArangoBackupPolicy, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(arangobackuppoliciesResource, "status", c.ns, arangoBackupPolicy), &backupv1.ArangoBackupPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*backupv1.ArangoBackupPolicy), err
}

// Delete takes name of the arangoBackupPolicy and deletes it. Returns an error if one occurs.
func (c *FakeArangoBackupPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(arangobackuppoliciesResource, c.ns, name), &backupv1.ArangoBackupPolicy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeArangoBackupPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(arangobackuppoliciesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &backupv1.ArangoBackupPolicyList{})
	return err
}

// Patch applies the patch and returns the patched arangoBackupPolicy.
func (c *FakeArangoBackupPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *backupv1.ArangoBackupPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(arangobackuppoliciesResource, c.ns, name, pt, data, subresources...), &backupv1.ArangoBackupPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*backupv1.ArangoBackupPolicy), err
}
