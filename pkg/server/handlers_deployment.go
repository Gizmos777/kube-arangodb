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

package server

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	api "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1alpha"
)

// Deployment is the API implemented by an ArangoDeployment.
type Deployment interface {
	Name() string
	Namespace() string
	Mode() api.DeploymentMode
	Environment() api.Environment
	PodCount() int
	ReadyPodCount() int
	VolumeCount() int
	ReadyVolumeCount() int
	StorageClasses() []string
	DatabaseURL() string
	DatabaseVersion() (string, string)
	Members() map[api.ServerGroup][]Member
}

// Member is the API implemented by a member of an ArangoDeployment.
type Member interface {
	ID() string
	PodName() string
	PVCName() string
	PVName() string
}

// DeploymentOperator is the API implemented by the deployment operator.
type DeploymentOperator interface {
	// GetDeployments returns basic information for all deployments managed by the operator
	GetDeployments() ([]Deployment, error)
	// GetDeployment returns detailed information for a deployment, managed by the operator, with given name
	GetDeployment(name string) (Deployment, error)
}

// DeploymentInfo is the information returned per deployment.
type DeploymentInfo struct {
	Name             string             `json:"name"`
	Namespace        string             `json:"namespace"`
	Mode             api.DeploymentMode `json:"mode"`
	Environment      api.Environment    `json:"environment"`
	PodCount         int                `json:"pod_count"`
	ReadyPodCount    int                `json:"ready_pod_count"`
	VolumeCount      int                `json:"volume_count"`
	ReadyVolumeCount int                `json:"ready_volume_count"`
	StorageClasses   []string           `json:"storage_classes"`
	DatabaseURL      string             `json:"database_url"`
	DatabaseVersion  string             `json:"database_version"`
	DatabaseLicense  string             `json:"database_license"`
}

// newDeploymentInfo initializes a DeploymentInfo for the given Deployment.
func newDeploymentInfo(d Deployment) DeploymentInfo {
	version, license := d.DatabaseVersion()
	return DeploymentInfo{
		Name:             d.Name(),
		Namespace:        d.Namespace(),
		Mode:             d.Mode(),
		Environment:      d.Environment(),
		PodCount:         d.PodCount(),
		ReadyPodCount:    d.ReadyPodCount(),
		VolumeCount:      d.VolumeCount(),
		ReadyVolumeCount: d.ReadyVolumeCount(),
		StorageClasses:   d.StorageClasses(),
		DatabaseURL:      d.DatabaseURL(),
		DatabaseVersion:  version,
		DatabaseLicense:  license,
	}
}

// MemberInfo contains detailed info of a specific member of the deployment
type MemberInfo struct {
	ID      string `json:"id"`
	PodName string `json:"pod_name"`
	PVCName string `json:"pvc_name"`
	PVName  string `json:"pv_name"`
}

// newMemberInfo creates a MemberInfo for the given member
func newMemberInfo(m Member) MemberInfo {
	return MemberInfo{
		ID:      m.ID(),
		PodName: m.PodName(),
		PVCName: m.PVCName(),
		PVName:  m.PVName(),
	}
}

// MemberGroupInfo contained detailed info of a group (e.g. Agent) of members
type MemberGroupInfo struct {
	Group   string       `json:"group"`
	Members []MemberInfo `json:"members"`
}

// DeploymentInfoDetails is the detailed information returned per deployment.
type DeploymentInfoDetails struct {
	DeploymentInfo
	MemberGroups []MemberGroupInfo `json:"member_groups"`
}

// newDeploymentInfoDetails initializes a DeploymentInfoDetails for the given Deployment.
func newDeploymentInfoDetails(d Deployment) DeploymentInfoDetails {
	result := DeploymentInfoDetails{
		DeploymentInfo: newDeploymentInfo(d),
	}
	for group, list := range d.Members() {
		memberInfos := make([]MemberInfo, len(list))
		for i, m := range list {
			memberInfos[i] = newMemberInfo(m)
		}
		result.MemberGroups = append(result.MemberGroups, MemberGroupInfo{
			Group:   strings.Title(group.AsRole()),
			Members: memberInfos,
		})
	}
	sort.Slice(result.MemberGroups, func(i, j int) bool {
		return result.MemberGroups[i].Group < result.MemberGroups[j].Group
	})
	return result
}

// Handle a GET /api/deployment request
func (s *Server) handleGetDeployments(c *gin.Context) {
	if do := s.deps.Operators.DeploymentOperator(); do != nil {
		// Fetch deployments
		depls, err := do.GetDeployments()
		if err != nil {
			sendError(c, err)
		} else {
			result := make([]DeploymentInfo, len(depls))
			for i, d := range depls {
				result[i] = newDeploymentInfo(d)
			}
			c.JSON(http.StatusOK, gin.H{
				"deployments": result,
			})
		}
	}
}

// Handle a GET /api/deployment/:name request
func (s *Server) handleGetDeploymentDetails(c *gin.Context) {
	if do := s.deps.Operators.DeploymentOperator(); do != nil {
		// Fetch deployments
		depl, err := do.GetDeployment(c.Params.ByName("name"))
		if err != nil {
			sendError(c, err)
		} else {
			result := newDeploymentInfoDetails(depl)
			c.JSON(http.StatusOK, result)
		}
	}
}
