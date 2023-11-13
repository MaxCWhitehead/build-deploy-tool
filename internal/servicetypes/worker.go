package servicetypes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var worker = ServiceType{
	Name: "worker",
	PrimaryContainer: ServiceContainer{
		Name:            "worker",
		ImagePullPolicy: corev1.PullAlways,
		Container: corev1.Container{
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/bin/sh",
							"-c",
							"if [ -x /bin/entrypoint-readiness ]; then /bin/entrypoint-readiness; fi",
						},
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       2,
				FailureThreshold:    3,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("10m"),
					corev1.ResourceMemory: resource.MustParse("100M"),
				},
			},
		},
	},
}

var workerPersistent = ServiceType{
	Name: "worker-persistent",
	PrimaryContainer: ServiceContainer{
		Name:            cli.PrimaryContainer.Name,
		ImagePullPolicy: cli.PrimaryContainer.ImagePullPolicy,
		Container:       cli.PrimaryContainer.Container,
	},
}
