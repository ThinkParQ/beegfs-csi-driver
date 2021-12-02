package controllers

import (
	"context"
	"fmt"
	"strconv"

	deploy "github.com/netapp/beegfs-csi-driver/deploy/k8s"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Integration tests using envtest", func() {
	var (
		ctx     context.Context
		cr      *beegfsv1.BeegfsDriver
		timeout = "5s"
	)

	BeforeEach(func() {
		By("Submitting a BeegfsDriver CR")
		cr = getValidCRWithAllFields()
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cr.Namespace}}
		ctx = context.Background()
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
	})

	Context("When a valid BeegfsDriver CR is submitted", func() {
		It("should create a correct Stateful Set", func() {
			sts, err := deploy.GetControllerServiceStatefulSet()
			Expect(err).NotTo(HaveOccurred())
			namespacedName := types.NamespacedName{Name: sts.Name, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, sts)
			}, timeout).Should(Succeed())
			Expect(sts.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))
			Expect(sts.Spec.Template.Annotations).To(HaveKey(annotationConfigMapVersion))
			Expect(sts.Spec.Template.Annotations).To(HaveKey(annotationConnauthSecretVersion))
		})

		It("should create a correct Daemon Set", func() {
			ds, err := deploy.GetNodeServiceDaemonSet()
			Expect(err).NotTo(HaveOccurred())
			namespacedName := types.NamespacedName{Name: ds.Name, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, ds)
			}, timeout).Should(Succeed())
			Expect(ds.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))
			Expect(ds.Spec.Template.Annotations).To(HaveKey(annotationConfigMapVersion))
			Expect(ds.Spec.Template.Annotations).To(HaveKey(annotationConnauthSecretVersion))
		})

		It("should create a correct Config Map", func() {
			cm := new(corev1.ConfigMap)
			namespacedName := types.NamespacedName{Name: deploy.ResourceNameConfigMap, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, cm)
			}, timeout).Should(Succeed())
			Expect(cm.ObjectMeta.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))
		})

		It("should create a correct Secret", func() {
			s := new(corev1.Secret)
			namespacedName := types.NamespacedName{Name: deploy.ResourceNameSecret, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, s)
			}, timeout).Should(Succeed())
			Expect(s.ObjectMeta.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))
		})

		It("should create correct RBAC objects", func() {
			resources, err := deploy.GetRBAC()
			Expect(err).NotTo(HaveOccurred())
			for _, resource := range resources {
				switch object := resource.(type) {

				case *corev1.ServiceAccount:
					sa := new(corev1.ServiceAccount)
					namespacedName := types.NamespacedName{Name: object.Name, Namespace: cr.Namespace}
					Eventually(func() error {
						return k8sClient.Get(ctx, namespacedName, sa)
					}, timeout).Should(Succeed(), "failed to find Service Account %s", object.Name)
					Expect(sa.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))

				case *rbacv1.Role:
					r := new(rbacv1.Role)
					namespacedName := types.NamespacedName{Name: object.Name, Namespace: cr.Namespace}
					Eventually(func() error {
						return k8sClient.Get(ctx, namespacedName, r)
					}, timeout).Should(Succeed(), "failed to find Role %s", object.Name)
					Expect(r.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))

				case *rbacv1.RoleBinding:
					rb := new(rbacv1.RoleBinding)
					namespacedName := types.NamespacedName{Name: object.Name, Namespace: cr.Namespace}
					Eventually(func() error {
						return k8sClient.Get(ctx, namespacedName, rb)
					}, timeout).Should(Succeed(), "failed to find Role Binding %s", object.Name)
					Expect(rb.OwnerReferences[0].Name).To(Equal("csi-beegfs-cr"))

				case *rbacv1.ClusterRole:
					clusterRole := new(rbacv1.ClusterRole)
					// Cluster-scoped resources have no namespace.
					namespacedName := types.NamespacedName{Name: object.Name}
					Eventually(func() error {
						return k8sClient.Get(ctx, namespacedName, clusterRole)
					}, timeout).Should(Succeed(), "failed to find Cluster Role %s", object.Name)
					// Cluster-scoped resources can't be owned by our CR.

				case *rbacv1.ClusterRoleBinding:
					crb := new(rbacv1.ClusterRoleBinding)
					// Cluster-scoped resources have no namespace.
					namespacedName := types.NamespacedName{Name: object.Name}
					Eventually(func() error {
						return k8sClient.Get(ctx, namespacedName, crb)
					}, timeout).Should(Succeed(), "failed to find Cluster Role Binding %s", object.Name)
					// Cluster-scoped resources can't be owned by our CR.

				default:
					Fail(fmt.Sprintf("Encountered unexpected type %T in RBAC manifests", object))
				}
			}
		})

		It("should create a correct CSI Driver", func() {
			driver, err := deploy.GetCSIDriver()
			Expect(err).NotTo(HaveOccurred())
			namespacedName := types.NamespacedName{Name: driver.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, driver)
			}, timeout).ShouldNot(HaveOccurred(), "failed to find CSI Driver %s", driver.Name)
		})

		It("should apply a finalizer", func() {
			Eventually(func() ([]string, error) {
				namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
				err := k8sClient.Get(ctx, namespacedName, cr)
				return cr.Finalizers, err
			}, timeout).Should(ContainElement(finalizerClusterResourceDeletion))
		})

		// Our status will never reflect readiness because there is no kube-controller-manager, kube-scheduler, or
		// nodes to run Pods.
		It("should have a status that correctly reflects unreadiness", func() {
			Eventually(func(g Gomega) {
				namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
				err := k8sClient.Get(ctx, namespacedName, cr)
				g.Expect(err).NotTo(HaveOccurred())
				csCondition := meta.FindStatusCondition(cr.Status.Conditions, beegfsv1.ConditionControllerServiceReady)
				g.Expect(csCondition).ToNot(BeNil())
				g.Expect(csCondition.Reason).To(Equal(beegfsv1.ReasonPodsNotScheduled))
				nsCondition := meta.FindStatusCondition(cr.Status.Conditions, beegfsv1.ConditionNodeServiceReady)
				g.Expect(nsCondition).ToNot(BeNil())
				g.Expect(nsCondition.Reason).To(Equal(beegfsv1.ReasonPodsNotScheduled))
			}, timeout).Should(Succeed())
		})
	})

	Context("When a BeegfsDriver CR is deleted", func() {
		BeforeEach(func() {
			By("Waiting for the expected finalizer to exist")
			Eventually(func() ([]string, error) {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
				return cr.Finalizers, err
			}, timeout).Should(ContainElement(finalizerClusterResourceDeletion))

			By("Deleting the BeegfsDriver CR")
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
		})

		It("should clean up cluster-scoped resources", func() {
			resources, err := deploy.GetRBAC()
			Expect(err).NotTo(HaveOccurred())
			for _, resource := range resources {
				switch object := resource.(type) {
				case *rbacv1.ClusterRole:
					Eventually(func() error {
						return k8sClient.Get(ctx, types.NamespacedName{Name: object.Name}, &rbacv1.ClusterRole{})
					}, timeout).ShouldNot(Succeed(), "expected Cluster Role %s to be deleted", object.Name)
				case rbacv1.ClusterRoleBinding:
					Eventually(func() error {
						return k8sClient.Get(ctx, types.NamespacedName{Name: object.Name}, &rbacv1.ClusterRoleBinding{})
					}, timeout).ShouldNot(Succeed(), "expected Cluster Role Binding %s to be deleted", object.Name)
				}
			}

			var driver *storagev1.CSIDriver
			driver, err = deploy.GetCSIDriver()
			Expect(err).NotTo(HaveOccurred())
			namespacedName := types.NamespacedName{Name: driver.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, driver)
			}, timeout).ShouldNot(Succeed(), "expected CSI Driver object to be deleted")
		})
	})

	Context("When a resource is modified", func() {
		var (
			cm  *corev1.ConfigMap
			s   *corev1.Secret
			sts *appsv1.StatefulSet
			ds  *appsv1.DaemonSet
		)

		BeforeEach(func() {
			var err error

			By("Retrieving the old resources")
			cm = new(corev1.ConfigMap)
			namespacedName := types.NamespacedName{Name: deploy.ResourceNameConfigMap, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, cm)
			}, timeout).Should(Succeed())

			s = new(corev1.Secret)
			namespacedName = types.NamespacedName{Name: deploy.ResourceNameSecret, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, s)
			}, timeout).Should(Succeed())

			sts, err = deploy.GetControllerServiceStatefulSet()
			Expect(err).NotTo(HaveOccurred())
			namespacedName = types.NamespacedName{Name: sts.Name, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, sts)
			}, timeout).Should(Succeed())

			ds, err = deploy.GetNodeServiceDaemonSet()
			Expect(err).NotTo(HaveOccurred())
			namespacedName = types.NamespacedName{Name: ds.Name, Namespace: cr.Namespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, ds)
			}, timeout).Should(Succeed())
		})

		Context("When the pluginConfig section of the CR is modified", func() {
			BeforeEach(func() {
				By("Submitting a modified BeegfsDriver CR")
				namespacedName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
				Eventually(func() error {
					var err error
					if err = k8sClient.Get(ctx, namespacedName, cr); err == nil {
						cr.Spec.PluginConfigFromFile.DefaultConfig.BeegfsClientConf["key"] = "newValue"
						return k8sClient.Update(ctx, cr)
					}
					return err
				}, timeout).Should(Succeed())
			})

			It("should update the Config Map", func() {
				oldResourceVersion := cm.ResourceVersion
				namespacedName := types.NamespacedName{Name: deploy.ResourceNameConfigMap, Namespace: cr.Namespace}
				Eventually(func() (string, error) {
					err := k8sClient.Get(ctx, namespacedName, cm)
					return cm.ResourceVersion, err
				}, timeout).ShouldNot(Equal(oldResourceVersion))
				Expect(cm.Data[deploy.KeyNameConfigMap]).To(ContainSubstring("newValue"))
			})

			It("should update the Stateful Set", func() {
				oldResourceVersion := sts.ResourceVersion
				namespacedName := types.NamespacedName{Name: sts.Name, Namespace: cr.Namespace}
				Eventually(func() (string, error) {
					err := k8sClient.Get(ctx, namespacedName, sts)
					return sts.ResourceVersion, err
				}, timeout).ShouldNot(Equal(oldResourceVersion))
			})

			It("should update the Daemon Set", func() {
				oldResourceVersion := ds.ResourceVersion
				namespacedName := types.NamespacedName{Name: ds.Name, Namespace: cr.Namespace}
				Eventually(func() (string, error) {
					err := k8sClient.Get(ctx, namespacedName, ds)
					return ds.ResourceVersion, err
				}, timeout).ShouldNot(Equal(oldResourceVersion))
			})
		})

		Context("When the Secret is modified", func() {
			BeforeEach(func() {
				By("Submitting a modified Secret")
				namespacedName := types.NamespacedName{Name: s.Name, Namespace: cr.Namespace}
				var err error
				Eventually(func() error {
					if err = k8sClient.Get(ctx, namespacedName, s); err == nil {
						s.StringData = map[string]string{deploy.KeyNameSecret: "secret"}
						return k8sClient.Update(ctx, s)
					}
					return err
				}, timeout).Should(Succeed())
			})

			It("should update the Stateful Set", func() {
				namespacedName := types.NamespacedName{Name: sts.Name, Namespace: cr.Namespace}
				Eventually(func() (string, error) {
					err := k8sClient.Get(ctx, namespacedName, sts)
					return sts.Spec.Template.Annotations[annotationConnauthSecretVersion], err
				}, timeout).Should(ContainSubstring(s.ResourceVersion))
			})

			It("should update the Daemon Set", func() {
				namespacedName := types.NamespacedName{Name: ds.Name, Namespace: cr.Namespace}
				Eventually(func() (string, error) {
					err := k8sClient.Get(ctx, namespacedName, ds)
					return ds.Spec.Template.Annotations[annotationConnauthSecretVersion], err
				}, timeout).Should(ContainSubstring(s.ResourceVersion))
			})
		})
	})

	Context("When an invalid BeegfsDriver CR is submitted", func() {
		// We can NOT test the inability to create a BeegfsDriver CR that is NOT named csi-beegfs-cr because we
		// bootstrap the test environment from the config/crd/bases/ directory, but the restriction is created in the
		// config/crd/patches/ directory and applied via Kustomize. This would be prohibitively difficult to replicate
		// in our test environment.

		Context("When the BeegfsDriver CR is a duplicate", func() {
			It("should fail", func() {
				// A duplicate csi-beegfs-cr was created in BeforeEach.
				cr.ResourceVersion = ""
				err := k8sClient.Create(ctx, cr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})
		})

		Context("When the log level is too high", func() {
			It("should fail", func() {
				logLevel := 6
				cr.Spec.LogLevel = &logLevel
				err := k8sClient.Create(ctx, cr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("less than or equal to 5"))
			})
		})
	})
})

var _ = Describe("Unit tests of helper functions", func() {
	Describe("setImages", func() {
		var containers []corev1.Container

		BeforeEach(func() {
			containers = []corev1.Container{
				{Name: deploy.ContainerNameCsiProvisioner, Image: "default.domain/default-provisioner-image:default-provisioner-tag"},
				{Name: deploy.ContainerNameBeegfsCsiDriver, Image: "default.domain/default-driver-image:default-driver-tag"},
				{Name: deploy.ContainerNameLivenessProbe, Image: "default.domain/default-liveness-image:default-liveness-tag"},
				{Name: deploy.ContainerNameCsiNodeDriverRegistrar, Image: "default.domain/default-registrar-image:default-registrar-tag"},
			}
		})

		Context("When there are overrides for all containers", func() {
			It("should override all images", func() {
				// If everything works correctly, these overrides take precedence over default images.
				overrides := beegfsv1.ContainerImageOverrides{
					BeegfsCsiDriver:        beegfsv1.ContainerImageOverride{Image: "override.domain/override-driver", Tag: "override-tag"},
					CsiNodeDriverRegistrar: beegfsv1.ContainerImageOverride{Image: "override.domain/override-registrar", Tag: "override-tag"},
					CsiProvisioner:         beegfsv1.ContainerImageOverride{Image: "override.domain/override-provisioner", Tag: "override-tag"},
					LivenessProbe:          beegfsv1.ContainerImageOverride{Image: "override.domain/override-liveness", Tag: "override-tag"},
				}
				setImages(ctrl.Log, containers, overrides)
				Expect(getContainerImageForName(deploy.ContainerNameCsiNodeDriverRegistrar, containers)).To(Equal("override.domain/override-registrar:override-tag"))
				Expect(getContainerImageForName(deploy.ContainerNameCsiProvisioner, containers)).To(Equal("override.domain/override-provisioner:override-tag"))
				Expect(getContainerImageForName(deploy.ContainerNameBeegfsCsiDriver, containers)).To(Equal("override.domain/override-driver:override-tag"))
				Expect(getContainerImageForName(deploy.ContainerNameLivenessProbe, containers)).To(Equal("override.domain/override-liveness:override-tag"))
			})
		})
	})

	Describe("getImageStringWithOverride", func() {
		Context("When override is empty", func() {
			It("should override nothing", func() {
				imageString := getImageStringWithOverride("default.domain/default-image:default-tag",
					beegfsv1.ContainerImageOverride{Image: "", Tag: ""})
				Expect(imageString).To(Equal("default.domain/default-image:default-tag"))
			})
		})

		Context("When only tag is overridden", func() {
			It("should only override tag", func() {
				imageString := getImageStringWithOverride("default.domain/default-image:default-tag",
					beegfsv1.ContainerImageOverride{Image: "", Tag: "override-tag"})
				Expect(imageString).To(Equal("default.domain/default-image:override-tag"))
			})
		})

		Context("When only image is overridden", func() {
			It("should only override image", func() {
				imageString := getImageStringWithOverride("default.domain/default-image:default-tag",
					beegfsv1.ContainerImageOverride{Image: "override.domain/override-image", Tag: ""})
				Expect(imageString).To(Equal("override.domain/override-image:default-tag"))
			})
		})

		Context("When both image and tag are overridden", func() {
			It("should override both image and tag", func() {
				imageString := getImageStringWithOverride("default.domain/default-image:default-tag",
					beegfsv1.ContainerImageOverride{Image: "override.domain/override-image", Tag: "override-tag"})
				Expect(imageString).To(Equal("override.domain/override-image:override-tag"))
			})
		})
	})

	Describe("containsString", func() {
		var slice = []string{"foo", "bar", "baz"}

		Context("When a slice contains a string", func() {
			It("should return true", func() {
				Expect(containsString(slice, "bar")).To(BeTrue())
			})
		})

		Context("When a slice does not contain a string", func() {
			It("should return false", func() {
				Expect(containsString(slice, "thud")).To(BeFalse())
			})
		})

		Context("When a slice is nil", func() {
			It("should return false", func() {
				slice = nil
				Expect(containsString(slice, "bar")).To(BeFalse())
			})
		})
	})

	Describe("setLogLevel", func() {
		var containers []corev1.Container

		BeforeEach(func() {
			containers = []corev1.Container{
				// Container with no environment variables.
				{},
				// Container with one environment variable.
				{Env: []corev1.EnvVar{
					{Name: "LOG_LEVEL", Value: "3"},
				}},
				// Container with multiple environment variables.
				{Env: []corev1.EnvVar{
					{Name: "SOME_OTHER_VARIABLE", Value: "we don't care"},
					{Name: "SOME_PREFIX_LOG_LEVEL", Value: "we don't care"},
					{Name: "LOG_LEVEL_SOME_SUFFIX", Value: "we don't care"},
					{Name: "LOG_LEVEL", Value: "3"},
				}},
			}
		})

		Context("When logLevel is nil", func() {
			It("should not error and should not change from default", func() {
				setLogLevel(ctrl.Log, nil, containers)
				for _, container := range containers {
					for _, envVar := range container.Env {
						if envVar.Name == "LOG_LEVEL" {
							Expect(envVar.Value).To(Equal("3"))
						}
					}
				}
			})

			It("should not modify other variables", func() {
				for _, container := range containers {
					for _, envVar := range container.Env {
						if envVar.Name != "LOG_LEVEL" {
							Expect(envVar.Value).To(Equal("we don't care"))
						}
					}
				}
			})
		})

		Context("When logLevel is set", func() {
			It("should change", func() {
				logLevel := 5
				setLogLevel(ctrl.Log, &logLevel, containers)
				for _, container := range containers {
					for _, envVar := range container.Env {
						if envVar.Name == "LOG_LEVEL" {
							Expect(envVar.Value).To(Equal("5"))
						}
					}
				}
			})

			It("should not modify other variables", func() {
				for _, container := range containers {
					for _, envVar := range container.Env {
						if envVar.Name != "LOG_LEVEL" {
							Expect(envVar.Value).To(Equal("we don't care"))
						}
					}
				}
			})
		})
	})
})

// getContainerImageForName is a helper function used only in tests. It returns the image field of a Container in a
// slice of Containers if its name field matches the passed name string. If no Container meets the criteria,
// getContainerImageForName returns "".
func getContainerImageForName(name string, containers []corev1.Container) string {
	for _, container := range containers {
		if container.Name == name {
			return container.Image
		}
	}
	return ""
}

// getValidCRWithNoFields is a helper function that returns a pointer to a BeegfsDriver CR in a random namespace with
// no configuration.
func getValidCRWithNoFields() *beegfsv1.BeegfsDriver {
	return &beegfsv1.BeegfsDriver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "csi-beegfs-cr",
			Namespace: fmt.Sprintf("test-%s", strconv.Itoa(rand.Intn(10000))), // Use a random namespace.
		},
	}
}

// getValidCRWithAllFields() is a helper function that returns a pointer to a BeegfsDriver CR in a random namespace
// with all configurable fields filled out.
func getValidCRWithAllFields() *beegfsv1.BeegfsDriver {
	cr := getValidCRWithNoFields()

	cr.Spec.NodeAffinityControllerService = corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "key",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"value"},
						},
					},
				},
			},
		},
	}

	cr.Spec.NodeAffinityNodeService = corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "key",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"value"},
						},
					},
				},
			},
		},
	}

	const (
		overrideImage = "some.registry/some/image"
		overrideTag   = "some-tag"
	)
	cr.Spec.ContainerImageOverrides = beegfsv1.ContainerImageOverrides{
		BeegfsCsiDriver: beegfsv1.ContainerImageOverride{
			Image: overrideImage,
			Tag:   overrideTag,
		},
		CsiNodeDriverRegistrar: beegfsv1.ContainerImageOverride{
			Image: overrideImage,
			Tag:   overrideTag,
		},
		CsiProvisioner: beegfsv1.ContainerImageOverride{
			Image: overrideImage,
			Tag:   overrideTag,
		},
		LivenessProbe: beegfsv1.ContainerImageOverride{
			Image: overrideImage,
			Tag:   overrideTag,
		},
	}

	beegfsConfig := beegfsv1.BeegfsConfig{
		ConnInterfaces:    []string{"interface1"},
		ConnNetFilter:     []string{"0.0.0.0"},
		ConnTcpOnlyFilter: []string{"0.0.0.0"},
		BeegfsClientConf:  map[string]string{"key": "value"},
	}
	filesystemSpecificConfigs := []beegfsv1.FileSystemSpecificConfig{
		{
			SysMgmtdHost: "0.0.0.0",
			Config:       beegfsConfig,
		},
	}
	cr.Spec.PluginConfigFromFile = beegfsv1.PluginConfigFromFile{
		PluginConfig: beegfsv1.PluginConfig{
			DefaultConfig:             beegfsConfig,
			FileSystemSpecificConfigs: filesystemSpecificConfigs,
		},
		NodeSpecificConfigs: []beegfsv1.NodeSpecificConfig{
			{
				NodeList:                  []string{"node1"},
				DefaultConfig:             beegfsConfig,
				FileSystemSpecificConfigs: filesystemSpecificConfigs,
			},
		},
	}

	logLevel := 3
	cr.Spec.LogLevel = &logLevel

	return cr
}
