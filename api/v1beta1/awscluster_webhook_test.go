/*
Copyright 2021 The Kubernetes Authors.

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

package v1beta1

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	utildefaulting "sigs.k8s.io/cluster-api/util/defaulting"
)

func TestAWSClusterDefault(t *testing.T) {
	cluster := &AWSCluster{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}
	t.Run("for AWSCluster", utildefaulting.DefaultValidateTest(cluster))
	cluster.Default()
	g := NewWithT(t)
	g.Expect(cluster.Spec.IdentityRef).NotTo(BeNil())
}

func TestAWSCluster_ValidateCreate(t *testing.T) {
	unsupportedIncorrectScheme := ClassicELBScheme("any-other-scheme")

	tests := []struct {
		name    string
		cluster *AWSCluster
		wantErr bool
		expect  func(g *WithT, res *AWSLoadBalancerSpec)
	}{
		// The SSHKeyName tests were moved to sshkeyname_test.go
		{
			name: "Default nil scheme to `internet-facing`",
			cluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			expect: func(g *WithT, res *AWSLoadBalancerSpec) {
				g.Expect(res.Scheme.String(), ClassicELBSchemeInternetFacing.String())
			},
			wantErr: false,
		},
		{
			name: "Internet-facing ELB scheme is defaulted to internet-facing during creation",
			cluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{Scheme: &ClassicELBSchemeIncorrectInternetFacing},
				},
			},
			expect: func(g *WithT, res *AWSLoadBalancerSpec) {
				g.Expect(res.Scheme.String(), ClassicELBSchemeInternetFacing.String())
			},
			wantErr: false,
		},
		{
			name: "Supported schemes are 'internet-facing, Internet-facing, internal, or nil', rest will be rejected",
			cluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{Scheme: &unsupportedIncorrectScheme},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid tags are rejected",
			cluster: &AWSCluster{
				Spec: AWSClusterSpec{
					AdditionalTags: Tags{
						"key-1":                    "value-1",
						"":                         "value-2",
						strings.Repeat("CAPI", 33): "value-3",
						"key-4":                    strings.Repeat("CAPI", 65),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := tt.cluster.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "cluster-",
				Namespace:    "default",
			}
			ctx := context.TODO()
			if err := testEnv.Create(ctx, cluster); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			c := &AWSCluster{}
			key := client.ObjectKey{
				Name:      cluster.Name,
				Namespace: "default",
			}

			g := NewWithT(t)
			g.Eventually(func() bool {
				err := testEnv.Get(ctx, key, c)
				return err == nil
			}, 10*time.Second).Should(Equal(true))

			tt.expect(g, c.Spec.ControlPlaneLoadBalancer)
		})
	}
}

func TestAWSCluster_ValidateUpdate(t *testing.T) {
	tests := []struct {
		name       string
		oldCluster *AWSCluster
		newCluster *AWSCluster
		wantErr    bool
	}{
		{
			name: "region is immutable",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					Region: "us-east-1",
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					Region: "us-east-2",
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneLoadBalancer name is immutable",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Name: aws.String("old-apiserver"),
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Name: aws.String("new-apiserver"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneLoadBalancer name is immutable, even if it is nil",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Name: nil,
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Name: aws.String("example-apiserver"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneLoadBalancer scheme is immutable",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Scheme: &ClassicELBSchemeInternal,
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Scheme: &ClassicELBSchemeInternetFacing,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneLoadBalancer scheme is immutable when left empty",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Scheme: &ClassicELBSchemeInternal,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneLoadBalancer scheme can be set to default when left empty",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Scheme: &ClassicELBSchemeInternetFacing,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "controlPlaneLoadBalancer crossZoneLoadBalancer is mutable",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						CrossZoneLoadBalancing: false,
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						CrossZoneLoadBalancing: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "controlPlaneEndpoint is immutable",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "example.com",
						Port: int32(8000),
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "foo.example.com",
						Port: int32(9000),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneEndpoint can be updated if it is empty",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: "example.com",
						Port: int32(8000),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "removal of externally managed annotation is not allowed",
			oldCluster: &AWSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{clusterv1.ManagedByAnnotation: ""},
				},
			},
			newCluster: &AWSCluster{},
			wantErr:    true,
		},
		{
			name:       "adding externally managed annotation is allowed",
			oldCluster: &AWSCluster{},
			newCluster: &AWSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{clusterv1.ManagedByAnnotation: ""},
				},
			},
			wantErr: false,
		},
		{
			name: "VPC id is immutable cannot be emptied once set",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: VPCSpec{ID: "managed-or-unmanaged-vpc"},
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			wantErr: true,
		},
		{
			name: "VPC id is immutable, cannot be set to a different value once set",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: VPCSpec{ID: "managed-or-unmanaged-vpc"},
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: VPCSpec{ID: "a-new-vpc"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid keys are not accepted during update",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					AdditionalTags: Tags{
						"key-1": "value-1",
						"key-2": "value-2",
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					AdditionalTags: Tags{
						"key-1":                    "value-1",
						"":                         "value-2",
						strings.Repeat("CAPI", 33): "value-3",
						"key-4":                    strings.Repeat("CAPI", 65),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail if controlPlaneLoadBalancer healthcheckprotocol is updated",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						HealthCheckProtocol: &ClassicELBProtocolTCP,
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						HealthCheckProtocol: &ClassicELBProtocolSSL,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should pass if controlPlaneLoadBalancer healthcheckprotocol is same after update",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						HealthCheckProtocol: &ClassicELBProtocolTCP,
					},
				},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						HealthCheckProtocol: &ClassicELBProtocolTCP,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should fail if controlPlaneLoadBalancer healthcheckprotocol is changed to non-default if it was not set before update",
			oldCluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			newCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						HealthCheckProtocol: &ClassicELBProtocolTCP,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			cluster := tt.oldCluster.DeepCopy()
			cluster.ObjectMeta.GenerateName = "cluster-"
			cluster.ObjectMeta.Namespace = "default"

			if err := testEnv.Create(ctx, cluster); err != nil {
				t.Errorf("failed to create cluster: %v", err)
			}
			cluster.ObjectMeta.Annotations = tt.newCluster.Annotations
			cluster.Spec = tt.newCluster.Spec
			if err := testEnv.Update(ctx, cluster); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		},
		)
	}
}

func TestAWSCluster_DefaultCNIIngressRules(t *testing.T) {
	AZUsageLimit := 3
	defaultVPCSpec := VPCSpec{
		AvailabilityZoneUsageLimit: &AZUsageLimit,
		AvailabilityZoneSelection:  &AZSelectionSchemeOrdered,
	}
	g := NewWithT(t)
	tests := []struct {
		name          string
		beforeCluster *AWSCluster
		afterCluster  *AWSCluster
	}{
		{
			name: "CNI ingressRules are updated cni spec undefined",
			beforeCluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			afterCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: defaultVPCSpec,
						CNI: &CNISpec{
							CNIIngressRules: CNIIngressRules{
								{
									Description: "bgp (calico)",
									Protocol:    SecurityGroupProtocolTCP,
									FromPort:    179,
									ToPort:      179,
								},
								{
									Description: "IP-in-IP (calico)",
									Protocol:    SecurityGroupProtocolIPinIP,
									FromPort:    -1,
									ToPort:      65535,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "CNIIngressRules are not added for empty CNISpec",
			beforeCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: defaultVPCSpec,
						CNI: &CNISpec{},
					},
				},
			},
			afterCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: defaultVPCSpec,
						CNI: &CNISpec{},
					},
				},
			},
		},
		{
			name: "CNI ingressRules are unmodified when they exist",
			beforeCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: defaultVPCSpec,
						CNI: &CNISpec{
							CNIIngressRules: CNIIngressRules{
								{
									Description: "Antrea 1",
									Protocol:    SecurityGroupProtocolTCP,
									FromPort:    10349,
									ToPort:      10349,
								},
							},
						},
					},
				},
			},
			afterCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					NetworkSpec: NetworkSpec{
						VPC: defaultVPCSpec,
						CNI: &CNISpec{
							CNIIngressRules: CNIIngressRules{
								{
									Description: "Antrea 1",
									Protocol:    SecurityGroupProtocolTCP,
									FromPort:    10349,
									ToPort:      10349,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			cluster := tt.beforeCluster.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "cluster-",
				Namespace:    "default",
			}
			g.Expect(testEnv.Create(ctx, cluster)).To(Succeed())
			g.Expect(cluster.Spec.NetworkSpec).To(Equal(tt.afterCluster.Spec.NetworkSpec))
		})
	}
}

func TestAWSCluster_ValidateAllowedCIDRBlocks(t *testing.T) {
	tests := []struct {
		name    string
		awsc    *AWSCluster
		wantErr bool
	}{
		{
			name: "allow valid CIDRs",
			awsc: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks: []string{
							"192.168.0.0/16",
							"192.168.0.1/32",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disableIngressRules allowed with empty CIDR block",
			awsc: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks:   []string{},
						DisableIngressRules: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disableIngressRules not allowed with CIDR blocks",
			awsc: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks: []string{
							"192.168.0.0/16",
							"192.168.0.1/32",
						},
						DisableIngressRules: true,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid CIDR block with invalid network",
			awsc: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks: []string{
							"100.200.300.400/99",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid CIDR block with garbage string",
			awsc: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks: []string{
							"abcdefg",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			cluster := tt.awsc.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "cluster-",
				Namespace:    "default",
			}
			if err := testEnv.Create(ctx, cluster); (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllowedCIDRBlocks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAWSCluster_DefaultAllowedCIDRBlocks(t *testing.T) {
	g := NewWithT(t)
	tests := []struct {
		name          string
		beforeCluster *AWSCluster
		afterCluster  *AWSCluster
	}{
		{
			name: "empty AllowedCIDRBlocks is defaulted to allow open ingress to bastion host",
			beforeCluster: &AWSCluster{
				Spec: AWSClusterSpec{},
			},
			afterCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks: []string{
							"0.0.0.0/0",
						},
					},
				},
			},
		},
		{
			name: "AllowedCIDRBlocks change not allowed if DisableIngressRules is true",
			beforeCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						AllowedCIDRBlocks:   []string{"0.0.0.0/0"},
						DisableIngressRules: true,
						Enabled:             true,
					},
				},
			},
			afterCluster: &AWSCluster{
				Spec: AWSClusterSpec{
					Bastion: Bastion{
						DisableIngressRules: true,
						Enabled:             true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			cluster := tt.beforeCluster.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "cluster-",
				Namespace:    "default",
			}
			err := testEnv.Create(ctx, cluster)
			if err != nil {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(cluster.Spec.Bastion).To(Equal(tt.afterCluster.Spec.Bastion))
			}
		})
	}
}
