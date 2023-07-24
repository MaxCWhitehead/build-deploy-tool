package generator

import (
	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/uselagoon/build-deploy-tool/internal/dbaasclient"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
)

// BuildValues is the values file data generated by the lagoon build
type BuildValues struct {
	Project              string             `json:"project"`
	Environment          string             `json:"environment"`
	EnvironmentType      string             `json:"environmentType"`
	Namespace            string             `json:"namespace"`
	GitSha               string             `json:"gitSha"`
	BuildType            string             `json:"buildType"`
	Kubernetes           string             `json:"kubernetes"`
	LagoonVersion        string             `json:"lagoonVersion"`
	ActiveEnvironment    string             `json:"activeEnvironment"`
	StandbyEnvironment   string             `json:"standbyEnvironment"`
	IsActiveEnvironment  bool               `json:"isActiveEnvironment"`
	IsStandbyEnvironment bool               `json:"isStandbyEnvironment"`
	PodSecurityContext   PodSecurityContext `json:"podSecurityContext"`
	ImagePullSecrets     []ImagePullSecrets `json:"imagePullSecrets"`
	Branch               string             `json:"branch"`
	PRNumber             string             `json:"prNumber"`
	PRTitle              string             `json:"prTitle"`
	PRHeadBranch         string             `json:"prHeadBranch"`
	PRBaseBranch         string             `json:"prBaseBranch"`
	Fastly               struct {
		ServiceID     string `json:"serviceId"`
		APISecretName string `json:"apiSecretName"`
		Watch         bool   `json:"watch"`
	} `json:"fastly"`
	FastlyCacheNoCache            string                      `json:"fastlyCacheNoCahce"`
	FastlyAPISecretPrefix         string                      `json:"fastlyAPISecretPrefix"`
	ConfigMapSha                  string                      `json:"configMapSha"`
	Route                         string                      `json:"route"`
	Routes                        []string                    `json:"routes"`
	AutogeneratedRoutes           []string                    `json:"autogeneratedRoutes"`
	RoutesAutogeneratePrefixes    []string                    `json:"routesAutogeneratePrefixes"`
	AutogeneratedRoutesFastly     bool                        `json:"autogeneratedRoutesFastly"`
	Services                      []ServiceValues             `json:"services"`
	Backup                        BackupConfiguration         `json:"backup"`
	Monitoring                    MonitoringConfig            `json:"monitoring"`
	DBaaSOperatorEndpoint         string                      `json:"dbaasOperatorEndpoint"`
	ServiceTypeOverrides          *lagoon.EnvironmentVariable `json:"serviceTypeOverrides"`
	DBaaSEnvironmentTypeOverrides *lagoon.EnvironmentVariable `json:"dbaasEnvironmentTypeOverrides"`
	DBaaSFallbackSingle           bool                        `json:"dbaasFallbackSingle"`
	IngressClass                  string                      `json:"ingressClass"`
	TaskScaleMaxIterations        int                         `json:"taskScaleMaxIterations"`
	TaskScaleWaitTime             int                         `json:"taskScaleWaitTime"`
	DynamicSecretMounts           []DynamicSecretMounts       `json:"dynamicSecretMounts"`
	DynamicSecretVolumes          []DynamicSecretVolumes      `json:"dynamicSecretVolumes"`
	DBaaSClient                   *dbaasclient.Client         `json:"-"`
	ImageReferences               map[string]string           `json:"imageReferences"`
	Resources                     Resources                   `json:"resources"`
	CronjobsDisabled              bool                        `json:"cronjobsDisabled"`
	Flags                         map[string]bool             `json:"-"`
}

type Resources struct {
	Limits   ResourceLimits   `json:"limits"`
	Requests ResourceRequests `json:"requests"`
}

type ResourceLimits struct {
	Memory           string `json:"memory"`
	EphemeralStorage string `json:"ephemeral-storage"`
}

type ResourceRequests struct {
	EphemeralStorage string `json:"ephemeral-storage"`
}

type PodSecurityContext struct {
	FsGroup    int64 `json:"fsGroup"`
	RunAsGroup int64 `json:"runAsGroup"`
	RunAsUser  int64 `json:"runAsUser"`
}

type MonitoringConfig struct {
	Enabled      bool   `json:"enabled"`
	AlertContact string `json:"alertContact"`
	StatusPageID string `json:"statusPageID"`
}

type ImagePullSecrets struct {
	Name string `json:"name"`
}

type DynamicSecretMounts struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

type DynamicSecretVolumes struct {
	Name   string        `json:"name"`
	Secret DynamicSecret `json:"secret"`
}

type DynamicSecret struct {
	SecretName string `json:"secretName"`
	Optional   bool   `json:"optional"`
}

// ServiceValues is the values for a specific service used by a lagoon build
type ServiceValues struct {
	Name                          string                  `json:"name"`         // this is the actual compose service name
	OverrideName                  string                  `json:"overrideName"` // if an override name is provided, use it
	Type                          string                  `json:"type"`
	AutogeneratedRoutesEnabled    bool                    `json:"autogeneratedRoutesEnabled"`
	AutogeneratedRoutesTLSAcme    bool                    `json:"autogeneratedRoutesTLSAcme"`
	AutogeneratedRouteDomain      string                  `json:"autogeneratedRouteDomain"`
	ShortAutogeneratedRouteDomain string                  `json:"shortAutogeneratedRouteDomain"`
	DBaaSEnvironment              string                  `json:"dbaasEnvironment"`
	NativeCronjobs                []lagoon.Cronjob        `json:"nativeCronjobs"`
	InPodCronjobs                 []lagoon.Cronjob        `json:"inPodCronjobs"`
	ImageName                     string                  `json:"imageName"`
	DeploymentServiceType         string                  `json:"deploymentServiceType"`
	ServicePort                   int32                   `json:"servicePort,omitempty"`
	PersistentVolumePath          string                  `json:"persistentVolumePath,omitempty"`
	PersistentVolumeName          string                  `json:"persistentVolumeName,omitempty"`
	PersistentVolumeSize          string                  `json:"persistentVolumeSize,omitempty"`
	UseSpotInstances              bool                    `json:"useSpot"`
	ForceSpotInstances            bool                    `json:"forceUseSpot"`
	CronjobUseSpotInstances       bool                    `json:"cronjobUseSpot"`
	CronjobForceSpotInstances     bool                    `json:"cronjobForceUseSpot"`
	Replicas                      int32                   `json:"replicas"`
	LinkedService                 *ServiceValues          `json:"linkedService"`
	PodSecurityContext            PodSecurityContext      `json:"podSecurityContext"`
	AdditionalServicePorts        []AdditionalServicePort `json:"additionalServicePorts,omitempty"`
}

type AdditionalServicePort struct {
	ServicePort composetypes.ServicePortConfig `json:"servicePort,omitempty"`
	ServiceName string                         `json:"serviceName,omitempty"`
}

// CronjobValues is the values for cronjobs
type CronjobValues struct {
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

type BackupConfiguration struct {
	PruneRetention PruneRetention              `json:"pruneRetention"`
	PruneSchedule  string                      `json:"pruneSchedule"`
	CheckSchedule  string                      `json:"checkSchedule"`
	BackupSchedule string                      `json:"backupSchedule"`
	S3Endpoint     string                      `json:"s3Endpoint"`
	S3BucketName   string                      `json:"s3BucketName"`
	S3SecretName   string                      `json:"s3SecretName"`
	CustomLocation CustomBackupRestoreLocation `json:"customLocation"`
}

type CustomBackupRestoreLocation struct {
	BackupLocationAccessKey  string `json:"backupLocationAccessKey"`
	BackupLocationSecretKey  string `json:"backupLocationSecretKey"`
	RestoreLocationAccessKey string `json:"restoreLocationAccessKey"`
	RestoreLocationSecretKey string `json:"restoreLocationSecretKey"`
}

type PruneRetention struct {
	Hourly  int `json:"hourly"`
	Daily   int `json:"daily"`
	Weekly  int `json:"weekly"`
	Monthly int `json:"monthly"`
}
