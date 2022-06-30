package kubernetes

import (
	"fmt"
	"path"

	"github.com/determined-ai/determined/master/pkg/etc"

	"github.com/docker/docker/api/types/mount"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

func configureMountPropagation(b *mount.BindOptions) *k8sV1.MountPropagationMode {
	if b == nil {
		return nil
	}

	switch b.Propagation {
	case mount.PropagationPrivate:
		p := k8sV1.MountPropagationNone
		return &p
	case mount.PropagationRSlave:
		p := k8sV1.MountPropagationHostToContainer
		return &p
	case mount.PropagationRShared:
		p := k8sV1.MountPropagationBidirectional
		return &p
	default:
		return nil
	}
}

func dockerMountsToHostVolumes(dockerMounts []mount.Mount) ([]k8sV1.VolumeMount, []k8sV1.Volume) {
	volumeMounts := make([]k8sV1.VolumeMount, 0, len(dockerMounts))
	volumes := make([]k8sV1.Volume, 0, len(dockerMounts))

	for idx, d := range dockerMounts {
		name := fmt.Sprintf("det-host-volume-%d", idx)
		volumeMounts = append(volumeMounts, k8sV1.VolumeMount{
			Name:             name,
			ReadOnly:         d.ReadOnly,
			MountPath:        d.Target,
			MountPropagation: configureMountPropagation(d.BindOptions),
		})
		volumes = append(volumes, k8sV1.Volume{
			Name: name,
			VolumeSource: k8sV1.VolumeSource{
				HostPath: &k8sV1.HostPathVolumeSource{
					Path: d.Source,
				},
			},
		})
	}

	return volumeMounts, volumes
}

func configureShmVolume(_ int64) (k8sV1.VolumeMount, k8sV1.Volume) {
	// Kubernetes does not support a native way to set shm size for
	// containers. The workaround for this is to create an emptyDir
	// volume and mount it to /dev/shm.
	volumeName := "det-shm-volume"
	volumeMount := k8sV1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  false,
		MountPath: "/dev/shm",
	}
	volume := k8sV1.Volume{
		Name: volumeName,
		VolumeSource: k8sV1.VolumeSource{EmptyDir: &k8sV1.EmptyDirVolumeSource{
			Medium: k8sV1.StorageMediumMemory,
		}},
	}
	return volumeMount, volume
}

func configureAdditionalFilesVolumes(
	configMapName string,
	runArchives []cproto.RunArchive,
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume) {
	initContainerVolumeMounts := make([]k8sV1.VolumeMount, 0)
	mainContainerVolumeMounts := make([]k8sV1.VolumeMount, 0)
	volumes := make([]k8sV1.Volume, 0)

	// In order to inject additional files into k8 pods, we un-tar the archives
	// in an initContainer from a configMap to an emptyDir, and then mount the
	// emptyDir into the main container.

	archiveVolumeName := "archive-volume"
	archiveVolume := k8sV1.Volume{
		Name: archiveVolumeName,
		VolumeSource: k8sV1.VolumeSource{
			ConfigMap: &k8sV1.ConfigMapVolumeSource{
				LocalObjectReference: k8sV1.LocalObjectReference{Name: configMapName},
			},
		},
	}
	volumes = append(volumes, archiveVolume)
	archiveVolumeMount := k8sV1.VolumeMount{
		Name:      archiveVolumeName,
		MountPath: initContainerTarSrcPath,
		ReadOnly:  true,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, archiveVolumeMount)

	entryPointVolumeName := "entrypoint-volume"
	var entryPointVolumeMode int32 = 0777 //0700 TODO this is bad.
	entryPointVolume := k8sV1.Volume{
		Name: entryPointVolumeName,
		VolumeSource: k8sV1.VolumeSource{
			ConfigMap: &k8sV1.ConfigMapVolumeSource{
				LocalObjectReference: k8sV1.LocalObjectReference{Name: configMapName},
				Items: []k8sV1.KeyToPath{{
					Key:  etc.K8InitContainerEntryScriptResource,
					Path: etc.K8InitContainerEntryScriptResource,
				}},
				DefaultMode: &entryPointVolumeMode,
			},
		},
	}
	volumes = append(volumes, entryPointVolume)
	entrypointVolumeMount := k8sV1.VolumeMount{
		Name:      entryPointVolumeName,
		MountPath: initContainerWorkDir,
		ReadOnly:  true,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, entrypointVolumeMount)

	additionalFilesVolumeName := "additional-files-volume"
	dstVolume := k8sV1.Volume{
		Name:         additionalFilesVolumeName,
		VolumeSource: k8sV1.VolumeSource{EmptyDir: &k8sV1.EmptyDirVolumeSource{}},
	}
	volumes = append(volumes, dstVolume)
	dstVolumeMount := k8sV1.VolumeMount{
		Name:      additionalFilesVolumeName,
		MountPath: initContainerTarDstPath,
		ReadOnly:  false,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, dstVolumeMount)

	rootPathsToItem := make(map[string][]archive.Item)
	for idx, runArchive := range runArchives {
		for _, item := range runArchive.Archive {
			// Files that aren't owned by root can get extracted.
			if item.UserID != 0 {
				mainContainerVolumeMounts = append(mainContainerVolumeMounts, k8sV1.VolumeMount{
					Name:      additionalFilesVolumeName,
					MountPath: path.Join(runArchive.Path, item.Path),
					SubPath:   path.Join(fmt.Sprintf("%d", idx), item.Path),
				})
			} else {
				dir := path.Dir(item.Path)
				rootPathsToItem[dir] = append(rootPathsToItem[dir], item)
			}
		}
	}

	// Files owned by root will be added as a config map unextracted.
	i := 0 // TODO
	for dir, items := range rootPathsToItem {
		i++ // TODO
		volumeName := fmt.Sprintf("root-file-%d", i)

		var keyToPaths []k8sV1.KeyToPath
		for _, item := range items {
			itemBase := path.Base(item.Path)
			mode := int32(item.FileMode)
			keyToPaths = append(keyToPaths, k8sV1.KeyToPath{
				Key:  itemBase,
				Path: itemBase,
				Mode: &mode,
			})
		}

		var entryPointVolumeMode int32 = 0777 //0700 TODO this is bad.
		volumes = append(volumes, k8sV1.Volume{
			Name: volumeName,
			VolumeSource: k8sV1.VolumeSource{
				ConfigMap: &k8sV1.ConfigMapVolumeSource{
					LocalObjectReference: k8sV1.LocalObjectReference{
						Name: configMapName,
					},
					Items:       keyToPaths,
					DefaultMode: &entryPointVolumeMode,
				},
			},
		})

		mainContainerVolumeMounts = append(mainContainerVolumeMounts, k8sV1.VolumeMount{
			Name:      volumeName,
			MountPath: dir,
			ReadOnly:  false, // TODO Assume root files are read only?
		})
	}

	return initContainerVolumeMounts, mainContainerVolumeMounts, volumes
}
