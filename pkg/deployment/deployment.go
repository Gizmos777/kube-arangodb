//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package deployment

import (
	"fmt"
	"reflect"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	api "github.com/arangodb/k8s-operator/pkg/apis/arangodb/v1alpha"
	"github.com/arangodb/k8s-operator/pkg/generated/clientset/versioned"
	"github.com/arangodb/k8s-operator/pkg/util/k8sutil"
	"github.com/arangodb/k8s-operator/pkg/util/retry"
	"github.com/arangodb/k8s-operator/pkg/util/trigger"
)

// Config holds configuration settings for a Deployment
type Config struct {
	ServiceAccount string
}

// Dependencies holds dependent services for a Deployment
type Dependencies struct {
	Log           zerolog.Logger
	KubeCli       kubernetes.Interface
	DatabaseCRCli versioned.Interface
}

// deploymentEventType strongly typed type of event
type deploymentEventType string

const (
	eventArangoDeploymentUpdated deploymentEventType = "ArangoDeploymentUpdated"
	eventPodAdded                deploymentEventType = "PodAdded"
	eventPodUpdated              deploymentEventType = "PodUpdated"
	eventPodDeleted              deploymentEventType = "PodDeleted"
)

// deploymentEvent holds an event passed from the controller to the deployment.
type deploymentEvent struct {
	Type       deploymentEventType
	Deployment *api.ArangoDeployment
	Pod        *v1.Pod
}

const (
	deploymentEventQueueSize = 100
	inspectionInterval       = time.Minute // Ensure we inspect the generated resources no less than with this interval
)

// Deployment is the in process state of an ArangoDeployment.
type Deployment struct {
	apiObject *api.ArangoDeployment // API object
	status    api.DeploymentStatus  // Internal status of the CR
	config    Config
	deps      Dependencies

	eventCh chan *deploymentEvent
	stopCh  chan struct{}

	eventsCli corev1.EventInterface

	inspectTrigger trigger.Trigger
}

// New creates a new Deployment from the given API object.
func New(config Config, deps Dependencies, apiObject *api.ArangoDeployment) (*Deployment, error) {
	if err := apiObject.Spec.Validate(); err != nil {
		return nil, maskAny(err)
	}
	d := &Deployment{
		apiObject: apiObject,
		status:    *(apiObject.Status.DeepCopy()),
		config:    config,
		deps:      deps,
		eventCh:   make(chan *deploymentEvent, deploymentEventQueueSize),
		stopCh:    make(chan struct{}),
		eventsCli: deps.KubeCli.Core().Events(apiObject.GetNamespace()),
	}

	go d.run()
	go d.listenForPodEvents()

	return d, nil
}

// Update the deployment.
// This sends an update event in the deployment event queue.
func (d *Deployment) Update(apiObject *api.ArangoDeployment) {
	d.send(&deploymentEvent{
		Type:       eventArangoDeploymentUpdated,
		Deployment: apiObject,
	})
}

// Delete the deployment.
// Called when the deployment was deleted by the user.
func (d *Deployment) Delete() {
	d.deps.Log.Info().Msg("deployment is deleted by user")
	close(d.stopCh)
}

// send given event into the deployment event queue.
func (d *Deployment) send(ev *deploymentEvent) {
	select {
	case d.eventCh <- ev:
		l, ecap := len(d.eventCh), cap(d.eventCh)
		if l > int(float64(ecap)*0.8) {
			d.deps.Log.Warn().
				Int("used", l).
				Int("capacity", ecap).
				Msg("event queue buffer is almost full")
		}
	case <-d.stopCh:
	}
}

// run is the core the core worker.
// It processes the event queue and polls the state of generated
// resource on a regular basis.
func (d *Deployment) run() {
	log := d.deps.Log

	// Create services
	if err := d.createServices(d.apiObject); err != nil {
		d.failOnError(err, "Failed to create services")
		return
	}

	// Create members
	if err := d.createInitialMembers(d.apiObject); err != nil {
		d.failOnError(err, "Failed to create initial members")
		return
	}

	// Create PVCs
	if err := d.ensurePVCs(d.apiObject); err != nil {
		d.failOnError(err, "Failed to create persistent volume claims")
		return
	}

	// Create pods
	if err := d.ensurePods(d.apiObject); err != nil {
		d.failOnError(err, "Failed to create pods")
		return
	}

	d.status.State = api.DeploymentStateRunning
	if err := d.updateCRStatus(); err != nil {
		log.Warn().Err(err).Msg("update initial CR status failed")
	}
	log.Info().Msg("start running...")

	for {
		select {
		case <-d.stopCh:
			// We're being stopped.
			return

		case event := <-d.eventCh:
			// Got event from event queue
			switch event.Type {
			case eventArangoDeploymentUpdated:
				if err := d.handleArangoDeploymentUpdatedEvent(event); err != nil {
					d.failOnError(err, "Failed to handle deployment update")
					return
				}
			case eventPodAdded, eventPodUpdated, eventPodDeleted:
				// Pod event received, let's inspect soon
				d.inspectTrigger.Trigger()
			default:
				panic("unknown event type" + event.Type)
			}

		case <-d.inspectTrigger.Done():
			// Inspection of generated resources needed
			if err := d.inspectPods(); err != nil {
				d.createEvent(k8sutil.NewErrorEvent("Pod inspection failed", err, d.apiObject))
			}
			// Ensure all resources are created
			if err := d.ensurePods(d.apiObject); err != nil {
				d.createEvent(k8sutil.NewErrorEvent("Pod creation failed", err, d.apiObject))
			}

		case <-time.After(inspectionInterval):
			// Trigger inspection
			d.inspectTrigger.Trigger()
		}
	}
}

// handleArangoDeploymentUpdatedEvent is called when the deployment is updated by the user.
func (d *Deployment) handleArangoDeploymentUpdatedEvent(event *deploymentEvent) error {
	// TODO
	return nil
}

// createEvent creates a given event.
// On error, the error is logged.
func (d *Deployment) createEvent(evt *v1.Event) {
	_, err := d.eventsCli.Create(evt)
	if err != nil {
		d.deps.Log.Error().Err(err).Interface("event", *evt).Msg("Failed to record event")
	}
}

// Update the status of the API object from the internal status
func (d *Deployment) updateCRStatus() error {
	if reflect.DeepEqual(d.apiObject.Status, d.status) {
		// Nothing has changed
		return nil
	}

	// Send update to API server
	update := d.apiObject.DeepCopy()
	update.Status = d.status
	newAPIObject, err := d.deps.DatabaseCRCli.DatabaseV1alpha().ArangoDeployments(d.apiObject.Namespace).Update(update)
	if err != nil {
		return maskAny(fmt.Errorf("failed to update ArangoDeployment status: %v", err))
	}

	// Update internal object
	d.apiObject = newAPIObject

	return nil
}

// failOnError reports the given error and sets the deployment status to failed.
func (d *Deployment) failOnError(err error, msg string) {
	log.Error().Err(err).Msg(msg)
	d.status.Reason = err.Error()
	d.reportFailedStatus()
}

// reportFailedStatus sets the status of the deployment to Failed and keeps trying to forward
// that to the API server.
func (d *Deployment) reportFailedStatus() {
	log := d.deps.Log
	log.Info().Msg("deployment failed. Reporting failed reason...")

	op := func() error {
		d.status.State = api.DeploymentStateFailed
		err := d.updateCRStatus()
		if err == nil || k8sutil.IsNotFound(err) {
			// Status has been updated
			return nil
		}

		if !k8sutil.IsConflict(err) {
			log.Warn().Err(err).Msg("retry report status: fail to update")
			return maskAny(err)
		}

		depl, err := d.deps.DatabaseCRCli.DatabaseV1alpha().ArangoDeployments(d.apiObject.Namespace).Get(d.apiObject.Name, metav1.GetOptions{})
		if err != nil {
			// Update (PUT) will return conflict even if object is deleted since we have UID set in object.
			// Because it will check UID first and return something like:
			// "Precondition failed: UID in precondition: 0xc42712c0f0, UID in object meta: ".
			if k8sutil.IsNotFound(err) {
				return nil
			}
			log.Warn().Err(err).Msg("retry report status: fail to get latest version")
			return maskAny(err)
		}
		d.apiObject = depl
		return maskAny(fmt.Errorf("retry needed"))
	}

	retry.Retry(op, time.Hour*24*365)
}

// isOwnerOf returns true if the given object belong to this deployment.
func (d *Deployment) isOwnerOf(obj metav1.Object) bool {
	ownerRefs := obj.GetOwnerReferences()
	if len(ownerRefs) < 1 {
		return false
	}
	return ownerRefs[0].UID == d.apiObject.UID
}