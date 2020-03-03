// +build !providerless

/*
Copyright 2016 The Kubernetes Authors.

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

package azure

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// This ensures load balancer exists and the frontend ip config is setup.
// This also reconciles the Service's Ports  with the LoadBalancer config.
// This entails adding rules/probes for expected Ports and removing stale rules/ports.
// nodes only used if wantLb is true
func (az *Cloud) reconcileLoadBalancerIPv6(clusterName string, service *v1.Service, nodes []*v1.Node, wantLb bool) (*network.LoadBalancer, error) {
	isInternal := requiresInternalLoadBalancer(service)
	serviceName := getServiceName(service)
	klog.V(2).Infof("reconcileLoadBalancerIPv6 for service(%s) - wantLb(%t): started", serviceName, wantLb)
	lb, _, _, err := az.getServiceLoadBalancer(service, clusterName, nodes, wantLb)
	if err != nil {
		klog.Errorf("reconcileLoadBalancerIPv6: failed to get load balancer for service %q, error: %v", serviceName, err)
		return nil, err
	}
	lbName := *lb.Name
	klog.V(2).Infof("reconcileLoadBalancerIPv6 for service(%s): lb(%s) wantLb(%t) resolved load balancer name", serviceName, lbName, wantLb)

	lbFrontendIPConfigName := az.getFrontendIPConfigName(service, subnet(service))
	lbFrontendIPConfigID := az.getFrontendIPConfigID(lbName, lbFrontendIPConfigName)
	lbv4FrontendIPConfigName := lbFrontendIPConfigName + "-v4"
	lbv4FrontendIPConfigID := az.getFrontendIPConfigID(lbName, lbv4FrontendIPConfigName)

	lbBackendPoolName := getBackendPoolName(clusterName, service)
	lbBackendPoolID := az.getBackendPoolID(lbName, lbBackendPoolName)

	lbIdleTimeout, err := getIdleTimeout(service)
	if wantLb && err != nil {
		return nil, err
	}

	dirtyLb := false

	// Ensure LoadBalancer's Backend Pool Configuration
	if wantLb {
		newBackendPools := []network.BackendAddressPool{}
		if lb.BackendAddressPools != nil {
			newBackendPools = *lb.BackendAddressPools
		}

		foundBackendPool := false
		for _, bp := range newBackendPools {
			if strings.EqualFold(*bp.Name, lbBackendPoolName) {
				klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb backendpool - found wanted backendpool. not adding anything", serviceName, wantLb)
				foundBackendPool = true
				break
			} else {
				klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb backendpool - found other backendpool %s", serviceName, wantLb, *bp.Name)
			}
		}
		if !foundBackendPool {
			newBackendPools = append(newBackendPools, network.BackendAddressPool{
				Name: to.StringPtr(lbBackendPoolName),
			})
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb backendpool - adding backendpool", serviceName, wantLb)

			dirtyLb = true
			lb.BackendAddressPools = &newBackendPools
		}
	}

	// Ensure LoadBalancer's Frontend IP Configurations
	dirtyConfigs := false
	newConfigs := []network.FrontendIPConfiguration{}
	if lb.FrontendIPConfigurations != nil {
		newConfigs = *lb.FrontendIPConfigurations
	}

	if !wantLb {
		for i := len(newConfigs) - 1; i >= 0; i-- {
			config := newConfigs[i]
			if az.serviceOwnsFrontendIP(config, service) {
				klog.V(2).Infof("reconcileLoadBalancer for service (%s)(%t): lb frontendconfig(%s) - dropping", serviceName, wantLb, *config.Name)
				newConfigs = append(newConfigs[:i], newConfigs[i+1:]...)
				dirtyConfigs = true
			}
		}
	} else {
		for i := len(newConfigs) - 1; i >= 0; i-- {
			config := newConfigs[i]
			// TODO(danmace): not clear why we would ever do isFrontendIPChanged check on a FIP not
			// owned by this service.
			if !az.serviceOwnsFrontendIP(config, service) {
				continue
			}
			// TODO(danmace): because isFrontendIPChanged assumes a 1:1 service to FIP ownership, passing
			// the config name is a way to bypass the check which causes isFrontendIPChanged to return
			// a diff when the name of the FIP has changed. It seems like the only way that's possible
			// is when the subnet (on which the name is based) changes. This seems like a very weird way
			// to do subnet diffing, and maybe I'm misunderstanding. For now, to avoid re-writing the
			// isFrontendIPChanged function, bypass that check. This might break subnet diffing, or whatever
			// that code is supposed to be accounting for.
			isFipChanged, err := az.isFrontendIPChanged(clusterName, config, service, *config.Name)
			if err != nil {
				return nil, err
			}
			if isFipChanged {
				klog.V(2).Infof("reconcileLoadBalancer for service (%s)(%t): lb frontendconfig(%s) - dropping", serviceName, wantLb, *config.Name)
				newConfigs = append(newConfigs[:i], newConfigs[i+1:]...)
				dirtyConfigs = true
			}
		}
		foundConfig := false
		for _, config := range newConfigs {
			if strings.EqualFold(*config.Name, lbFrontendIPConfigName) {
				foundConfig = true
				break
			}
		}
		if !foundConfig {
			pipName, shouldPIPExisted, err := az.determinePublicIPName(clusterName, service)
			if err != nil {
				return nil, err
			}
			domainNameLabel := getPublicIPDomainNameLabel(service)

			if ipv6 {
				pip4, err := az.ensurePublicIPExists(service, false, pipName+"-v4", domainNameLabel, clusterName, shouldPIPExisted)
				if err != nil {
					return nil, err
				}
				configProperties := &network.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress:         &network.PublicIPAddress{ID: pip4.ID},
					PrivateIPAddressVersion: "IPv4",
				}
				newConfigs = append(newConfigs,
					network.FrontendIPConfiguration{
						Name:                                    &lbv4FrontendIPConfigName,
						FrontendIPConfigurationPropertiesFormat: configProperties,
					})
			}

			pip, err := az.ensurePublicIPExists(service, ipv6, pipName, domainNameLabel, clusterName, shouldPIPExisted)
			if err != nil {
				return nil, err
			}
			configProperties := &network.FrontendIPConfigurationPropertiesFormat{
				PublicIPAddress:         &network.PublicIPAddress{ID: pip.ID},
				PrivateIPAddressVersion: "IPv6",
			}
			newConfigs = append(newConfigs,
				network.FrontendIPConfiguration{
					Name:                                    to.StringPtr(lbFrontendIPConfigName),
					FrontendIPConfigurationPropertiesFormat: configProperties,
				})
		}

		klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb frontendconfig(%s) - adding", serviceName, wantLb, lbFrontendIPConfigName)
		dirtyConfigs = true
	}
	if dirtyConfigs {
		dirtyLb = true
		lb.FrontendIPConfigurations = &newConfigs
	}

	// update probes/rules
	expectedProbes, expectedRules, err := az.reconcileLoadBalancerRule(service, wantLb, lbFrontendIPConfigID, lbBackendPoolID, lbName, lbIdleTimeout, "")
	if err != nil {
		return nil, err
	}
	if ipv6 {
		ipv4expectedProbes, ipv4expectedRules, err := az.reconcileLoadBalancerRule(service, wantLb, lbv4FrontendIPConfigID, lbBackendPoolID, lbName, lbIdleTimeout, "-v4")
		if err != nil {
			return nil, err
		}
		expectedProbes = append(expectedProbes, ipv4expectedProbes...)
		expectedRules = append(expectedRules, ipv4expectedRules...)
	}

	// remove unwanted probes
	dirtyProbes := false
	var updatedProbes []network.Probe
	if lb.Probes != nil {
		updatedProbes = *lb.Probes
	}
	for i := len(updatedProbes) - 1; i >= 0; i-- {
		existingProbe := updatedProbes[i]
		if az.serviceOwnsRule(service, *existingProbe.Name) {
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb probe(%s) - considering evicting", serviceName, wantLb, *existingProbe.Name)
			keepProbe := false
			if findProbe(expectedProbes, existingProbe) {
				klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb probe(%s) - keeping", serviceName, wantLb, *existingProbe.Name)
				keepProbe = true
			}
			if !keepProbe {
				updatedProbes = append(updatedProbes[:i], updatedProbes[i+1:]...)
				klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb probe(%s) - dropping", serviceName, wantLb, *existingProbe.Name)
				dirtyProbes = true
			}
		}
	}
	// add missing, wanted probes
	for _, expectedProbe := range expectedProbes {
		foundProbe := false
		if findProbe(updatedProbes, expectedProbe) {
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb probe(%s) - already exists", serviceName, wantLb, *expectedProbe.Name)
			foundProbe = true
		}
		if !foundProbe {
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb probe(%s) - adding", serviceName, wantLb, *expectedProbe.Name)
			updatedProbes = append(updatedProbes, expectedProbe)
			dirtyProbes = true
		}
	}
	if dirtyProbes {
		dirtyLb = true
		lb.Probes = &updatedProbes
	}

	// update rules
	dirtyRules := false
	var updatedRules []network.LoadBalancingRule
	if lb.LoadBalancingRules != nil {
		updatedRules = *lb.LoadBalancingRules
	}
	// update rules: remove unwanted
	for i := len(updatedRules) - 1; i >= 0; i-- {
		existingRule := updatedRules[i]
		if az.serviceOwnsRule(service, *existingRule.Name) {
			keepRule := false
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb rule(%s) - considering evicting", serviceName, wantLb, *existingRule.Name)
			if findRule(expectedRules, existingRule, wantLb) {
				klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb rule(%s) - keeping", serviceName, wantLb, *existingRule.Name)
				keepRule = true
			}
			if !keepRule {
				klog.V(2).Infof("reconcileLoadBalancer for service (%s)(%t): lb rule(%s) - dropping", serviceName, wantLb, *existingRule.Name)
				updatedRules = append(updatedRules[:i], updatedRules[i+1:]...)
				dirtyRules = true
			}
		}
	}
	// update rules: add needed
	for _, expectedRule := range expectedRules {
		foundRule := false
		if findRule(updatedRules, expectedRule, wantLb) {
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb rule(%s) - already exists", serviceName, wantLb, *expectedRule.Name)
			foundRule = true
		}
		if !foundRule {
			klog.V(10).Infof("reconcileLoadBalancer for service (%s)(%t): lb rule(%s) adding", serviceName, wantLb, *expectedRule.Name)
			updatedRules = append(updatedRules, expectedRule)
			dirtyRules = true
		}
	}
	if dirtyRules {
		dirtyLb = true
		lb.LoadBalancingRules = &updatedRules
	}

	// We don't care if the LB exists or not
	// We only care about if there is any change in the LB, which means dirtyLB
	// If it is not exist, and no change to that, we don't CreateOrUpdate LB
	if dirtyLb {
		if lb.FrontendIPConfigurations == nil || len(*lb.FrontendIPConfigurations) == 0 {
			// When FrontendIPConfigurations is empty, we need to delete the Azure load balancer resource itself,
			// because an Azure load balancer cannot have an empty FrontendIPConfigurations collection
			klog.V(2).Infof("reconcileLoadBalancer for service(%s): lb(%s) - deleting; no remaining frontendIPConfigurations", serviceName, lbName)

			// Remove backend pools from vmSets. This is required for virtual machine scale sets before removing the LB.
			vmSetName := az.mapLoadBalancerNameToVMSet(lbName, clusterName)
			klog.V(10).Infof("EnsureBackendPoolDeleted(%s,%s) for service %s: start", lbBackendPoolID, vmSetName, serviceName)
			err := az.vmSet.EnsureBackendPoolDeleted(service, lbBackendPoolID, vmSetName, lb.BackendAddressPools)
			if err != nil {
				klog.Errorf("EnsureBackendPoolDeleted(%s) for service %s failed: %v", lbBackendPoolID, serviceName, err)
				return nil, err
			}
			klog.V(10).Infof("EnsureBackendPoolDeleted(%s) for service %s: end", lbBackendPoolID, serviceName)

			// Remove the LB.
			klog.V(10).Infof("reconcileLoadBalancer: az.DeleteLB(%q): start", lbName)
			err = az.DeleteLB(service, lbName)
			if err != nil {
				klog.V(2).Infof("reconcileLoadBalancer for service(%s) abort backoff: lb(%s) - deleting; no remaining frontendIPConfigurations", serviceName, lbName)
				return nil, err
			}
			klog.V(10).Infof("az.DeleteLB(%q): end", lbName)
		} else {
			klog.V(2).Infof("reconcileLoadBalancer: reconcileLoadBalancer for service(%s): lb(%s) - updating", serviceName, lbName)
			err := az.CreateOrUpdateLB(service, *lb)
			if err != nil {
				klog.V(2).Infof("reconcileLoadBalancer for service(%s) abort backoff: lb(%s) - updating", serviceName, lbName)
				return nil, err
			}

			if isInternal {
				// Refresh updated lb which will be used later in other places.
				newLB, exist, err := az.getAzureLoadBalancer(lbName)
				if err != nil {
					klog.V(2).Infof("reconcileLoadBalancer for service(%s): getAzureLoadBalancer(%s) failed: %v", serviceName, lbName, err)
					return nil, err
				}
				if !exist {
					return nil, fmt.Errorf("load balancer %q not found", lbName)
				}
				lb = &newLB
			}
		}
	}

	if wantLb && nodes != nil {
		// Add the machines to the backend pool if they're not already
		vmSetName := az.mapLoadBalancerNameToVMSet(lbName, clusterName)
		// Etag would be changed when updating backend pools, so invalidate lbCache after it.
		defer az.lbCache.Delete(lbName)
		err := az.vmSet.EnsureHostsInPool(service, nodes, lbBackendPoolID, vmSetName, isInternal)
		if err != nil {
			return nil, err
		}
	}

	klog.V(2).Infof("reconcileLoadBalancer for service(%s): lb(%s) finished", serviceName, lbName)
	return lb, nil
}
