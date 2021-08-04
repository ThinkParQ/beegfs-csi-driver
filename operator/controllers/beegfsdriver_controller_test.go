package controllers

import (
	"github.com/netapp/beegfs-csi-driver/deploy"
	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Basic controller unit tests", func() {
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
