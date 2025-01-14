package imageprovider

import (
	"fmt"

	"github.com/go-logr/logr"

	metal3 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/metal3-io/baremetal-operator/pkg/imageprovider"
	"github.com/openshift/image-customization-controller/pkg/env"
	"github.com/openshift/image-customization-controller/pkg/ignition"
	"github.com/openshift/image-customization-controller/pkg/imagehandler"
)

type rhcosImageProvider struct {
	ImageHandler   imagehandler.ImageHandler
	EnvInputs      *env.EnvInputs
	RegistriesConf []byte
}

func NewRHCOSImageProvider(imageServer imagehandler.ImageHandler, inputs *env.EnvInputs) imageprovider.ImageProvider {
	registries, err := inputs.RegistriesConf()
	if err != nil {
		panic(err)
	}

	return &rhcosImageProvider{
		ImageHandler:   imageServer,
		EnvInputs:      inputs,
		RegistriesConf: registries,
	}
}

func (ip *rhcosImageProvider) SupportsArchitecture(arch string) bool {
	return true
}

func (ip *rhcosImageProvider) SupportsFormat(format metal3.ImageFormat) bool {
	switch format {
	case metal3.ImageFormatISO, metal3.ImageFormatInitRD:
		return true
	default:
		return false
	}
}

func (ip *rhcosImageProvider) buildIgnitionConfig(networkData imageprovider.NetworkData) ([]byte, error) {
	nmstateData := networkData["nmstate"]

	return ignition.New(nmstateData, ip.RegistriesConf,
		ip.EnvInputs.IronicBaseURL,
		ip.EnvInputs.IronicAgentImage,
		ip.EnvInputs.IronicAgentPullSecret,
		ip.EnvInputs.IronicRAMDiskSSHKey,
	).Generate()
}

func imageKey(data imageprovider.ImageData) string {
	return fmt.Sprintf("%s-%s-%s-%s.%s",
		data.ImageMetadata.Namespace,
		data.ImageMetadata.Name,
		data.ImageMetadata.UID,
		data.Architecture,
		data.Format,
	)
}

func (ip *rhcosImageProvider) BuildImage(data imageprovider.ImageData, networkData imageprovider.NetworkData, log logr.Logger) (string, error) {
	ignitionConfig, err := ip.buildIgnitionConfig(networkData)
	if err != nil {
		return "", err
	}

	return ip.ImageHandler.ServeImage(imageKey(data), ignitionConfig,
		data.Format == metal3.ImageFormatInitRD, false)
}

func (ip *rhcosImageProvider) DiscardImage(data imageprovider.ImageData) error {
	ip.ImageHandler.RemoveImage(imageKey(data))
	return nil
}
