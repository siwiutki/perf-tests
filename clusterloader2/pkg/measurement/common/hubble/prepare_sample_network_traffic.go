/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hubble

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/perf-tests/clusterloader2/pkg/framework"
	"k8s.io/perf-tests/clusterloader2/pkg/framework/client"
	"k8s.io/perf-tests/clusterloader2/pkg/measurement"
	"k8s.io/perf-tests/clusterloader2/pkg/util"
)

const (
	prepareSampleNetworkTrafficMeasurementName = "PrepareSampleNetworkTraffic"
	resourceReadyTimeout                       = 5 * time.Minute
	prepareSampleNetworkTrafficNamespace       = "sample-traffic"

	echoServerDeploymentPath = "manifests/echoserver-deployment.yaml"
	echoServerServicePath    = "manifests/echoserver-service.yaml"
	wrkDeploymentPath        = "manifests/wrk-deployment.yaml"
)

func init() {
	klog.Info("Registering Prepare Sample Network Traffic Measurement")
	if err := measurement.Register(prepareSampleNetworkTrafficMeasurementName,
		createPrepareSampleNetworkTrafficMeasurement); err != nil {
		klog.Fatalf("Cannot register %s: %w", prepareSampleNetworkTrafficMeasurementName, err)
	}
}

func createPrepareSampleNetworkTrafficMeasurement() measurement.Measurement {
	return &prepareSampleNetworkTrafficMeasurement{}
}

type prepareSampleNetworkTrafficMeasurement struct {
	framework *framework.Framework
	k8sClient kubernetes.Interface
}

func (p *prepareSampleNetworkTrafficMeasurement) Execute(config *measurement.Config) ([]measurement.Summary, error) {
	action, err := util.GetString(config.Params, "action")
	if err != nil {
		return nil, err
	}
	switch action {
	case "start":
		return nil, p.createObjects(config)
	case "gather":
		// TODO(siwy): Documentation states that `Dispose` is called when measurement
		//  is not used anymore. Not sure yet if `Dispose` can be triggered during test,
		//  so we have dummy "gather" action here to mark that measurement is good to be disposed.
		//  Ideally, we could get rid of "action" param entirely, needs checking
		//  if we can do that.
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown action %v", action)
	}
	return nil, nil
}

func (p *prepareSampleNetworkTrafficMeasurement) initialize(config *measurement.Config) {
	p.framework = config.ClusterFramework
	p.k8sClient = config.ClusterFramework.GetClientSets().GetClient()
}

func (p *prepareSampleNetworkTrafficMeasurement) createObjects(config *measurement.Config) error {
	p.initialize(config)
	if err := client.CreateNamespace(p.k8sClient, prepareSampleNetworkTrafficNamespace); err != nil {
		return fmt.Errorf("error while creating namesapce: %w", err)
	}
	if err := p.framework.ApplyTemplatedManifests(manifestsFS, echoServerDeploymentPath, nil); err != nil {
		return fmt.Errorf("error while creating echoserver deployment: %w", err)
	}
	if err := p.framework.ApplyTemplatedManifests(manifestsFS, echoServerServicePath, nil); err != nil {
		return fmt.Errorf("error while creating echoserver service: %w", err)
	}
	if err := p.framework.ApplyTemplatedManifests(manifestsFS, wrkDeploymentPath, nil); err != nil {
		return fmt.Errorf("error while creating wrk deployment: %w", err)
	}
	return nil
}

func (p *prepareSampleNetworkTrafficMeasurement) Dispose() {
	if p.framework == nil {
		klog.V(1).Infof("%s measurement wasn't started, skipping the Dispose() step", p.String())
		return
	}
	if err := client.DeleteNamespace(p.k8sClient, prepareSampleNetworkTrafficNamespace); err != nil {
		klog.Errorf("Failed to delete namespace %s: %v", prepareSampleNetworkTrafficNamespace, err)
	}
	if err := client.WaitForDeleteNamespace(p.k8sClient, prepareSampleNetworkTrafficNamespace, 2*time.Minute); err != nil {
		klog.Errorf("Waiting for namespace %s deletion failed: %v", prepareSampleNetworkTrafficNamespace, err)
	}
}

func (p *prepareSampleNetworkTrafficMeasurement) String() string {
	return prepareSampleNetworkTrafficMeasurementName
}
