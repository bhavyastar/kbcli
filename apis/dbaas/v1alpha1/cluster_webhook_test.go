/*
Copyright ApeCloud, Inc.

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

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("cluster webhook", func() {
	var (
		randomStr               = testCtx.GetRandomStr()
		clusterName             = "cluster-webhook-mysql-" + randomStr
		rsClusterName           = "cluster-webhook-rs-" + randomStr
		clusterDefinitionName   = "cluster-webhook-mysql-definition-" + randomStr
		rsClusterDefinitionName = "cluster-webhook-rs-definition-" + randomStr
		secondClusterDefinition = "cluster-webhook-mysql-definition2-" + randomStr
		clusterVersionName      = "cluster-webhook-mysql-clusterversion-" + randomStr
		rsClusterVersionName    = "cluster-webhook-rs-clusterversion-" + randomStr
		timeout                 = time.Second * 10
		interval                = time.Second
	)
	cleanupObjects := func() {
		// Add any setup steps that needs to be executed before each test
		err := k8sClient.DeleteAllOf(ctx, &Cluster{}, client.InNamespace(testCtx.DefaultNamespace), client.HasLabels{testCtx.TestObjLabelKey})
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &ClusterVersion{}, client.HasLabels{testCtx.TestObjLabelKey})
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &ClusterDefinition{}, client.HasLabels{testCtx.TestObjLabelKey})
		Expect(err).NotTo(HaveOccurred())
	}
	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
		cleanupObjects()
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
		cleanupObjects()
	})

	Context("When cluster create and update", func() {
		It("Should webhook validate passed", func() {
			By("By testing creating a new clusterDefinition when no clusterVersion and clusterDefinition")
			cluster, _ := createTestCluster(clusterDefinitionName, clusterVersionName, clusterName)
			Expect(testCtx.CreateObj(ctx, cluster).Error()).To(ContainSubstring("not found"))

			By("By creating a new clusterDefinition")
			clusterDef, _ := createTestClusterDefinitionObj(clusterDefinitionName)
			Expect(testCtx.CreateObj(ctx, clusterDef)).Should(Succeed())

			clusterDefSecond, _ := createTestClusterDefinitionObj(secondClusterDefinition)
			Expect(testCtx.CreateObj(ctx, clusterDefSecond)).Should(Succeed())

			// wait until ClusterDefinition created
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterDefinitionName}, clusterDef)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By creating a new clusterVersion")
			clusterVersion := createTestClusterVersionObj(clusterDefinitionName, clusterVersionName)
			Expect(testCtx.CreateObj(ctx, clusterVersion)).Should(Succeed())
			// wait until ClusterVersion created
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterVersionName}, clusterVersion)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By creating a new Cluster")
			cluster, _ = createTestCluster(clusterDefinitionName, clusterVersionName, clusterName)
			Expect(testCtx.CreateObj(ctx, cluster)).Should(Succeed())

			By("By testing update spec.clusterDefinitionRef")
			patch := client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.ClusterDefRef = secondClusterDefinition
			Expect(k8sClient.Patch(ctx, cluster, patch).Error()).To(ContainSubstring("spec.clusterDefinitionRef"))
			// restore
			cluster.Spec.ClusterDefRef = clusterDefinitionName

			By("By testing spec.components[?].type not found in clusterDefinitionRef")
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].Type = "replicaset"
			Expect(k8sClient.Patch(ctx, cluster, patch).Error()).To(ContainSubstring("is not found in ClusterDefinition.spec.components[*].typeName"))
			// restore
			cluster.Spec.Components[0].Type = "replicasets"

			// restore
			cluster.Spec.Components[0].Name = "replicasets"

			By("By updating spec.components[?].volumeClaimTemplates storage size, expect succeed")
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("2Gi")
			Expect(k8sClient.Patch(ctx, cluster, patch)).Should(Succeed())

			By("By updating spec.components[?].volumeClaimTemplates[?].name, expect not succeed")
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].VolumeClaimTemplates[0].Name = "test"
			Expect(k8sClient.Patch(ctx, cluster, patch).Error()).To(ContainSubstring("volumeClaimTemplates is forbidden modification except for storage size."))

			By("By updating component resources")
			// restore test volume claim template name to data
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].VolumeClaimTemplates[0].Name = "data"
			cluster.Spec.Components[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("200Mi"),
				},
			}
			Expect(k8sClient.Patch(ctx, cluster, patch)).Should(Succeed())
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":     resource.MustParse("100m"),
					"memory1": resource.MustParse("200Mi"),
				},
			}
			Expect(k8sClient.Patch(ctx, cluster, patch).Error()).To(ContainSubstring("resource key is not cpu or memory or hugepages- "))
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("100Mi"),
				},
			}
			Expect(k8sClient.Patch(ctx, cluster, patch).Error()).To(ContainSubstring("must be less than or equal to memory limit"))
			patch = client.MergeFrom(cluster.DeepCopy())
			cluster.Spec.Components[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse("80Mi")
			Expect(k8sClient.Patch(ctx, cluster, patch)).Should(Succeed())

			// A series of tests on clusters when componentType=replication
			By("By creating a new clusterDefinition when componentType=replication")
			rsClusterDef, _ := createTestReplicationSetClusterDefinitionObj(rsClusterDefinitionName)
			Expect(testCtx.CreateObj(ctx, rsClusterDef)).Should(Succeed())

			By("By creating a new clusterVersion when componentType=replication")
			rsClusterVersion := createTestReplicationSetClusterVersionObj(rsClusterDefinitionName, rsClusterVersionName)
			Expect(testCtx.CreateObj(ctx, rsClusterVersion)).Should(Succeed())

			By("By creating a new cluster when componentType=replication")
			rsCluster, _ := createTestReplicationSetCluster(rsClusterDefinitionName, rsClusterVersionName, rsClusterName)
			Expect(testCtx.CreateObj(ctx, rsCluster)).Should(Succeed())

			By("By updating cluster.Spec.Components[0].PrimaryIndex larger than cluster.Spec.Components[0].Replicas, expect not succeed")
			patch = client.MergeFrom(cluster.DeepCopy())
			*rsCluster.Spec.Components[0].PrimaryIndex = int32(3)
			*rsCluster.Spec.Components[0].Replicas = int32(3)
			Expect(k8sClient.Patch(ctx, rsCluster, patch)).ShouldNot(Succeed())

			By("By updating cluster.Spec.Components[0].PrimaryIndex less than cluster.Spec.Components[0].Replicas, expect succeed")
			patch = client.MergeFrom(cluster.DeepCopy())
			*rsCluster.Spec.Components[0].PrimaryIndex = int32(1)
			*rsCluster.Spec.Components[0].Replicas = int32(2)
			Expect(k8sClient.Patch(ctx, rsCluster, patch)).Should(Succeed())

			By("By updating cluster.Spec.Components[0].PrimaryIndex less than 0, expect not succeed")
			patch = client.MergeFrom(cluster.DeepCopy())
			*rsCluster.Spec.Components[0].PrimaryIndex = int32(-1)
			*rsCluster.Spec.Components[0].Replicas = int32(2)
			Expect(k8sClient.Patch(ctx, rsCluster, patch)).ShouldNot(Succeed())
		})
	})
})

func createTestCluster(clusterDefinitionName, clusterVersionName, clusterName string) (*Cluster, error) {
	clusterYaml := fmt.Sprintf(`
apiVersion: dbaas.kubeblocks.io/v1alpha1
kind: Cluster
metadata:
  name: %s
  namespace: default
spec:
  clusterDefinitionRef: %s
  clusterVersionRef: %s
  components:
  - name: replicasets
    type: replicasets
    replicas: 1
    volumeClaimTemplates: 
    - name: data
      spec:
        resources:
          requests:
            storage: 1Gi
  - name: proxy
    type: proxy
    replicas: 1
`, clusterName, clusterDefinitionName, clusterVersionName)
	cluster := &Cluster{}
	err := yaml.Unmarshal([]byte(clusterYaml), cluster)
	cluster.Spec.TerminationPolicy = WipeOut
	return cluster, err
}

func createTestReplicationSetCluster(clusterDefinitionName, clusterVerisonName, clusterName string) (*Cluster, error) {
	clusterYaml := fmt.Sprintf(`
apiVersion: dbaas.kubeblocks.io/v1alpha1
kind: Cluster
metadata:
  name: %s
  namespace: default
spec:
  clusterDefinitionRef: %s
  clusterVersionRef: %s
  components:
  - name: replication
    type: replication
    monitor: false
    primaryIndex: 0
    replicas: 2
    volumeClaimTemplates:
    - name: data
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
`, clusterName, clusterDefinitionName, clusterVerisonName)
	cluster := &Cluster{}
	err := yaml.Unmarshal([]byte(clusterYaml), cluster)
	cluster.Spec.TerminationPolicy = WipeOut
	return cluster, err
}
