package generator

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/uselagoon/build-deploy-tool/internal/dbaasclient"
	"github.com/uselagoon/build-deploy-tool/internal/helpers"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

type Generator struct {
	LagoonYAML                 *lagoon.YAML
	BuildValues                *BuildValues
	LagoonEnvironmentVariables *[]lagoon.EnvironmentVariable
	ActiveEnvironment          *bool
	StandbyEnvironment         *bool
	AutogeneratedRoutes        *lagoon.RoutesV2
	MainRoutes                 *lagoon.RoutesV2
	ActiveStandbyRoutes        *lagoon.RoutesV2
}

type GeneratorInput struct {
	LagoonYAML               string
	LagoonYAMLOverride       string
	LagoonVersion            string
	ProjectName              string
	EnvironmentName          string
	EnvironmentType          string
	ActiveEnvironment        string
	StandbyEnvironment       string
	ProjectVariables         string
	EnvironmentVariables     string
	BuildType                string
	Branch                   string
	PRNumber                 string
	PRTitle                  string
	PRHeadBranch             string
	PRBaseBranch             string
	MonitoringContact        string
	MonitoringStatusPageID   string
	FastlyCacheNoCahce       string
	FastlyAPISecretPrefix    string
	SavedTemplatesPath       string
	ConfigMapSha             string
	BackupConfiguration      BackupConfiguration
	IgnoreNonStringKeyErrors bool
	IgnoreMissingEnvFiles    bool
	Debug                    bool
	DBaaSClient              *dbaasclient.Client
	ImageReferences          map[string]string
	ImagePullSecret          string
	Namespace                string
	DefaultBackupSchedule    string
}

func NewGenerator(
	generator GeneratorInput,
) (*Generator, error) {

	// create some initial variables to be passed through the generators
	buildValues := BuildValues{}
	buildValues.Flags = map[string]bool{}
	lYAML := &lagoon.YAML{}
	lagoonEnvVars := []lagoon.EnvironmentVariable{}
	autogenRoutes := &lagoon.RoutesV2{}
	mainRoutes := &lagoon.RoutesV2{}
	activeStandbyRoutes := &lagoon.RoutesV2{}

	// environment variables will override what is provided by flags
	// the following variables have been identified as used by custom-ingress objects
	// these are available within a lagoon build as standard
	monitoringContact := helpers.GetEnv("MONITORING_ALERTCONTACT", generator.MonitoringContact, generator.Debug)
	monitoringStatusPageID := helpers.GetEnv("MONITORING_STATUSPAGEID", generator.MonitoringStatusPageID, generator.Debug)
	projectName := helpers.GetEnv("PROJECT", generator.ProjectName, generator.Debug)
	environmentName := helpers.GetEnv("ENVIRONMENT", generator.EnvironmentName, generator.Debug)
	branch := helpers.GetEnv("BRANCH", generator.Branch, generator.Debug)
	prNumber := helpers.GetEnv("PR_NUMBER", generator.PRNumber, generator.Debug)
	prTitle := helpers.GetEnv("PR_NUMBER", generator.PRTitle, generator.Debug)
	prHeadBranch := helpers.GetEnv("PR_HEAD_BRANCH", generator.PRHeadBranch, generator.Debug)
	prBaseBranch := helpers.GetEnv("PR_BASE_BRANCH", generator.PRBaseBranch, generator.Debug)
	environmentType := helpers.GetEnv("ENVIRONMENT_TYPE", generator.EnvironmentType, generator.Debug)
	buildType := helpers.GetEnv("BUILD_TYPE", generator.BuildType, generator.Debug)
	activeEnvironment := helpers.GetEnv("ACTIVE_ENVIRONMENT", generator.ActiveEnvironment, generator.Debug)
	standbyEnvironment := helpers.GetEnv("STANDBY_ENVIRONMENT", generator.StandbyEnvironment, generator.Debug)
	fastlyCacheNoCahce := helpers.GetEnv("LAGOON_FASTLY_NOCACHE_SERVICE_ID", generator.FastlyCacheNoCahce, generator.Debug)
	fastlyAPISecretPrefix := helpers.GetEnv("ROUTE_FASTLY_SERVICE_ID", generator.FastlyAPISecretPrefix, generator.Debug)
	lagoonVersion := helpers.GetEnv("LAGOON_VERSION", generator.LagoonVersion, generator.Debug)
	configMapSha := helpers.GetEnv("CONFIG_MAP_SHA", generator.ConfigMapSha, generator.Debug)

	buildValues.ConfigMapSha = configMapSha
	// get the image references values from the build images output
	buildValues.ImageReferences = generator.ImageReferences
	// add standard lagoon imagepull secret name
	buildValues.ImagePullSecrets = append(buildValues.ImagePullSecrets, ImagePullSecrets{Name: "lagoon-internal-registry-secret"})

	defaultBackupSchedule := helpers.GetEnv("DEFAULT_BACKUP_SCHEDULE", generator.DefaultBackupSchedule, generator.Debug)
	if defaultBackupSchedule == "" {
		defaultBackupSchedule = "M H(22-2) * * *"
	}

	// try source the namespace from the generator, but whatever is defined in the service account location
	// should be used if one exists, falls back to whatever came in via generator
	namespace := helpers.GetEnv("NAMESPACE", generator.Namespace, generator.Debug)
	namespace, err := helpers.GetNamespace(namespace, "/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		// a file was found, but there was an issue accessing it
		return nil, err
	}

	buildValues.Backup.K8upVersion = helpers.GetEnv("K8UP_VERSION", generator.BackupConfiguration.K8upVersion, generator.Debug)

	// get the project and environment variables
	projectVariables := helpers.GetEnv("LAGOON_PROJECT_VARIABLES", generator.ProjectVariables, generator.Debug)
	environmentVariables := helpers.GetEnv("LAGOON_ENVIRONMENT_VARIABLES", generator.EnvironmentVariables, generator.Debug)

	// read the .lagoon.yml file and the LAGOON_YAML_OVERRIDE if set
	if err := LoadAndUnmarshalLagoonYml(generator.LagoonYAML, generator.LagoonYAMLOverride, "LAGOON_YAML_OVERRIDE", lYAML, projectName, generator.Debug); err != nil {
		return nil, err
	}

	//add the dbaas client to build values too
	buildValues.DBaaSClient = generator.DBaaSClient

	buildValues.DefaultBackupSchedule = defaultBackupSchedule

	// set the task scale iterations/wait times
	// these are not user modifiable flags, but are injectable by the controller so individual clusters can
	// set these on their `remote-controller` deployments to be injected to builds.
	buildValues.TaskScaleMaxIterations = helpers.GetEnvInt("LAGOON_FEATURE_FLAG_TASK_SCALE_MAX_ITERATIONS", 30, generator.Debug)
	buildValues.TaskScaleWaitTime = helpers.GetEnvInt("LAGOON_FEATURE_FLAG_TASK_SCALE_WAIT_TIME", 10, generator.Debug)

	// start saving values into the build values variable
	buildValues.Project = projectName
	buildValues.Environment = environmentName
	buildValues.Namespace = namespace
	buildValues.EnvironmentType = environmentType
	buildValues.BuildType = buildType
	buildValues.LagoonVersion = lagoonVersion
	buildValues.ActiveEnvironment = activeEnvironment
	buildValues.StandbyEnvironment = standbyEnvironment
	buildValues.FastlyCacheNoCache = fastlyCacheNoCahce
	buildValues.FastlyAPISecretPrefix = fastlyAPISecretPrefix
	switch buildType {
	case "branch", "promote":
		buildValues.Branch = branch
	case "pullrequest":
		buildValues.PRNumber = prNumber
		buildValues.PRTitle = prTitle
		buildValues.PRHeadBranch = prHeadBranch
		buildValues.PRBaseBranch = prBaseBranch
		// since pullrequests don't  have a branch
		// we should set the branch to be `pr-PRNUMBER` so that it can be used for matching elsewhere where matching for `branch`
		// using buildvalues is done
		buildValues.Branch = fmt.Sprintf("pr-%v", prNumber)
	}

	// break out of the generator if these requirements are missing
	if projectName == "" || environmentName == "" || environmentType == "" || buildType == "" {
		return nil, fmt.Errorf("Missing arguments: project-name, environment-name, environment-type, or build-type not defined")
	}
	switch buildType {
	case "branch", "promote":
		if branch == "" {
			return nil, fmt.Errorf("Missing arguments: branch not defined")
		}
	case "pullrequest":
		if prNumber == "" || prHeadBranch == "" || prBaseBranch == "" {
			return nil, fmt.Errorf("Missing arguments: pullrequest-number, pullrequest-head-branch, or pullrequest-base-branch not defined")
		}
	}

	// get the dbaas operator http endpoint or fall back to the default
	buildValues.DBaaSOperatorEndpoint = helpers.GetEnv("DBAAS_OPERATOR_HTTP", "http://dbaas.lagoon.svc:5000", generator.Debug)

	// by default, environment routes are not monitored
	buildValues.Monitoring.Enabled = false
	if environmentType == "production" {
		// if this is a production environment, monitoring IS enabled
		buildValues.Monitoring.Enabled = true
		buildValues.Monitoring.AlertContact = monitoringContact
		buildValues.Monitoring.StatusPageID = monitoringStatusPageID
		// check if the environment is active or standby
		if environmentName == activeEnvironment {
			buildValues.IsActiveEnvironment = true
		}
		if environmentName == standbyEnvironment {
			buildValues.IsStandbyEnvironment = true
		}
	}

	// unmarshal and then merge the two so there is only 1 set of variables to iterate over
	projectVars := []lagoon.EnvironmentVariable{}
	envVars := []lagoon.EnvironmentVariable{}
	json.Unmarshal([]byte(projectVariables), &projectVars)
	json.Unmarshal([]byte(environmentVariables), &envVars)
	mergedVariables := lagoon.MergeVariables(projectVars, envVars)
	// collect a bunch of the default LAGOON_X based build variables that are injected into `lagoon-env` and make them available
	configVars := collectBuildVariables(buildValues)
	// add the calculated build runtime variables into the existing variable slice
	// this will later be used to add `runtime|global` scope into the `lagoon-env` configmap
	lagoonEnvVars = lagoon.MergeVariables(mergedVariables, configVars)

	imageCache := CheckFeatureFlag("IMAGECACHE_REGISTRY", lagoonEnvVars, generator.Debug)
	if imageCache != "" {
		if imageCache[len(imageCache)-1:] != "/" {
			imageCache = fmt.Sprintf("%s/", imageCache)
		}
	}
	buildValues.ImageCache = imageCache

	// check the environment for INGRESS_CLASS flag, will be "" if there are none found
	ingressClass := CheckFeatureFlag("INGRESS_CLASS", lagoonEnvVars, generator.Debug)
	buildValues.IngressClass = ingressClass

	// check for rootless workloads
	rootlessWorkloads := CheckFeatureFlag("ROOTLESS_WORKLOAD", lagoonEnvVars, generator.Debug)
	if rootlessWorkloads == "enabled" {
		buildValues.Flags["rootlessworkloads"] = true
		buildValues.PodSecurityContext = PodSecurityContext{
			RunAsGroup: 0,
			RunAsUser:  10000,
			FsGroup:    10001,
		}
	}

	fsOnRootMismatch := CheckFeatureFlag("FS_ON_ROOT_MISMATCH", lagoonEnvVars, generator.Debug)
	if fsOnRootMismatch == "enabled" {
		buildValues.PodSecurityContext.OnRootMismatch = true
	}

	// check admin features for resources
	buildValues.Resources.Limits.Memory = CheckAdminFeatureFlag("CONTAINER_MEMORY_LIMIT", false)
	buildValues.Resources.Limits.EphemeralStorage = CheckAdminFeatureFlag("EPHEMERAL_STORAGE_LIMIT", false)
	buildValues.Resources.Requests.EphemeralStorage = CheckAdminFeatureFlag("EPHEMERAL_STORAGE_REQUESTS", false)
	// validate that what is provided
	if buildValues.Resources.Limits.Memory != "" {
		err := ValidateResourceQuantity(buildValues.Resources.Limits.Memory)
		if err != nil {
			return nil, fmt.Errorf("provided memory limit %s is not a valid resource quantity", buildValues.Resources.Limits.Memory)
		}
	}
	if buildValues.Resources.Limits.EphemeralStorage != "" {
		err := ValidateResourceQuantity(buildValues.Resources.Limits.EphemeralStorage)
		if err != nil {
			return nil, fmt.Errorf("provided ephemeral storage limit %s is not a valid resource quantity", buildValues.Resources.Limits.EphemeralStorage)
		}
	}
	if buildValues.Resources.Requests.EphemeralStorage != "" {
		err := ValidateResourceQuantity(buildValues.Resources.Requests.EphemeralStorage)
		if err != nil {
			return nil, fmt.Errorf("provided  ephemeral storage requests %s is not a valid resource quantity", buildValues.Resources.Requests.EphemeralStorage)
		}
	}

	// get any variables from the API here
	lagoonServiceTypes, _ := lagoon.GetLagoonVariable("LAGOON_SERVICE_TYPES", nil, lagoonEnvVars)
	buildValues.ServiceTypeOverrides = lagoonServiceTypes

	lagoonDBaaSEnvironmentTypes, _ := lagoon.GetLagoonVariable("LAGOON_DBAAS_ENVIRONMENT_TYPES", nil, lagoonEnvVars)
	buildValues.DBaaSEnvironmentTypeOverrides = lagoonDBaaSEnvironmentTypes

	// check autogenerated routes for fastly `LAGOON_FEATURE_FLAG(_FORCE|_DEFAULT)_FASTLY_AUTOGENERATED` using feature flags
	autogeneratedRoutesFastly := CheckFeatureFlag("FASTLY_AUTOGENERATED", lagoonEnvVars, generator.Debug)
	if autogeneratedRoutesFastly == "enabled" {
		buildValues.AutogeneratedRoutesFastly = true
	} else {
		buildValues.AutogeneratedRoutesFastly = false
	}
	// check legacy variable in envvars
	lagoonAutogeneratedFastly, _ := lagoon.GetLagoonVariable("LAGOON_FASTLY_AUTOGENERATED", nil, lagoonEnvVars)
	if lagoonAutogeneratedFastly != nil {
		if lagoonAutogeneratedFastly.Value == "enabled" {
			buildValues.AutogeneratedRoutesFastly = true
		} else {
			buildValues.AutogeneratedRoutesFastly = false
		}
	}
	// check legacy variable in envvars
	cronjobsDisabled, _ := lagoon.GetLagoonVariable("LAGOON_CRONJOBS_DISABLED", nil, lagoonEnvVars)
	if cronjobsDisabled != nil {
		if cronjobsDisabled.Value == "true" {
			buildValues.CronjobsDisabled = true
		} else {
			buildValues.CronjobsDisabled = false
		}
	}

	// @TODO: eventually fail builds if this is not set https://github.com/uselagoon/build-deploy-tool/issues/56
	// lagoonDBaaSFallbackSingle, _ := lagoon.GetLagoonVariable("LAGOON_FEATURE_FLAG_DBAAS_FALLBACK_SINGLE", nil, lagoonEnvVars)
	// buildValues.DBaaSFallbackSingle = helpers.StrToBool(lagoonDBaaSFallbackSingle.Value)

	/* start backups configuration */
	err = generateBackupValues(&buildValues, lYAML, lagoonEnvVars, generator.Debug)
	if err != nil {
		return nil, err
	}
	/* end backups configuration */

	/* start compose->service configuration */
	err = generateServicesFromDockerCompose(&buildValues, lYAML, lagoonEnvVars, generator.IgnoreNonStringKeyErrors, generator.IgnoreMissingEnvFiles, generator.Debug)
	if err != nil {
		return nil, err
	}
	/* end compose->service configuration */

	/* start route generation */
	// create all the routes for this environment and store the primary and secondary routes into values
	// populate the autogenRoutes, mainRoutes and activeStandbyRoutes here and load them
	buildValues.Route, buildValues.Routes, buildValues.AutogeneratedRoutes, err = generateRoutes(
		lagoonEnvVars,
		buildValues,
		*lYAML,
		autogenRoutes,
		mainRoutes,
		activeStandbyRoutes,
		generator.Debug,
	)
	if err != nil {
		return nil, err
	}
	/* end route generation configuration */

	// finally return the generator values
	return &Generator{
		BuildValues:                &buildValues,
		LagoonYAML:                 lYAML,
		LagoonEnvironmentVariables: &lagoonEnvVars,
		ActiveEnvironment:          &buildValues.IsActiveEnvironment,
		StandbyEnvironment:         &buildValues.IsStandbyEnvironment,
		AutogeneratedRoutes:        autogenRoutes,
		MainRoutes:                 mainRoutes,
		ActiveStandbyRoutes:        activeStandbyRoutes,
	}, nil
}

func LoadAndUnmarshalLagoonYml(lagoonYml string, lagoonYmlOverride string, lagoonYmlOverrideEnvVarName string, lYAML *lagoon.YAML, projectName string, debug bool) error {

	// First we load the primary file
	if err := lagoon.UnmarshalLagoonYAML(lagoonYml, lYAML, projectName); err != nil {
		return fmt.Errorf("couldn't unmarshal file %v: %v", lagoonYml, err)
	}

	// Here we try and merge in .lagoon.yml override
	if _, err := os.Stat(lagoonYmlOverride); err == nil {
		overLagoonYaml := &lagoon.YAML{}
		if err := lagoon.UnmarshalLagoonYAML(lagoonYmlOverride, overLagoonYaml, projectName); err != nil {
			return fmt.Errorf("couldn't unmarshal file %v: %v", lagoonYmlOverride, err)
		}
		//now we merge
		if err := lagoon.MergeLagoonYAMLs(lYAML, overLagoonYaml); err != nil {
			return fmt.Errorf("unable to merge %v over %v: %v", lagoonYmlOverride, lagoonYml, err)
		}
	}
	// Now we see if there are any environment vars set for .lagoon.yml overrides
	envLagoonYamlStringBase64 := helpers.GetEnv(lagoonYmlOverrideEnvVarName, "", debug)
	if envLagoonYamlStringBase64 != "" {
		//Decode it
		envLagoonYamlString, err := base64.StdEncoding.DecodeString(envLagoonYamlStringBase64)
		if err != nil {
			return fmt.Errorf("Unable to decode %v - is it base64 encoded?", lagoonYmlOverrideEnvVarName)
		}
		envLagoonYaml := &lagoon.YAML{}
		lEnvLagoonPolysite := make(map[string]interface{})

		err = yaml.Unmarshal(envLagoonYamlString, envLagoonYaml)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal env var %v: %v", lagoonYmlOverrideEnvVarName, err)
		}
		err = yaml.Unmarshal(envLagoonYamlString, lEnvLagoonPolysite)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal env var %v: %v", lagoonYmlOverrideEnvVarName, err)
		}

		if _, ok := lEnvLagoonPolysite[projectName]; ok {
			s, _ := yaml.Marshal(lEnvLagoonPolysite[projectName])
			_ = yaml.Unmarshal(s, &envLagoonYaml)
		}
		//now we merge
		if err := lagoon.MergeLagoonYAMLs(lYAML, envLagoonYaml); err != nil {
			return fmt.Errorf("unable to merge LAGOON_YAML_OVERRIDE over %v: %v", lagoonYml, err)
		}
	}
	return nil
}

// this creates a bunch of standard environment variables that are injected into the `lagoon-env` configmap normally
func collectBuildVariables(buildValues BuildValues) []lagoon.EnvironmentVariable {
	vars := []lagoon.EnvironmentVariable{}
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_PROJECT", Value: buildValues.Project, Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_ENVIRONMENT", Value: buildValues.Environment, Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_ENVIRONMENT_TYPE", Value: buildValues.EnvironmentType, Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_GIT_SHA", Value: buildValues.GitSha, Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_KUBERNETES", Value: buildValues.Kubernetes, Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_GIT_SAFE_BRANCH", Value: buildValues.Environment, Scope: "runtime"}) //deprecated??? (https://github.com/uselagoon/lagoon/blob/1053965321495213591f4c9110f90a9d9dcfc946/images/kubectl-build-deploy-dind/build-deploy-docker-compose.sh#L748)
	if buildValues.BuildType == "branch" {
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_GIT_BRANCH", Value: buildValues.Branch, Scope: "runtime"})
	}
	if buildValues.BuildType == "pullrequest" {
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_PR_HEAD_BRANCH", Value: buildValues.PRHeadBranch, Scope: "runtime"})
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_PR_BASE_BRANCH", Value: buildValues.PRBaseBranch, Scope: "runtime"})
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_PR_TITLE", Value: buildValues.PRTitle, Scope: "runtime"})
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_PR_NUMBER", Value: buildValues.PRNumber, Scope: "runtime"})
	}
	if buildValues.ActiveEnvironment != "" {
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_ACTIVE_ENVIRONMENT", Value: buildValues.ActiveEnvironment, Scope: "runtime"})
	}
	if buildValues.StandbyEnvironment != "" {
		vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_STANDBY_ENVIRONMENT", Value: buildValues.StandbyEnvironment, Scope: "runtime"})
	}
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_ROUTE", Value: buildValues.Route, Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_ROUTES", Value: strings.Join(buildValues.Routes, ","), Scope: "runtime"})
	vars = append(vars, lagoon.EnvironmentVariable{Name: "LAGOON_AUTOGENERATED_ROUTES", Value: strings.Join(buildValues.AutogeneratedRoutes, ","), Scope: "runtime"})
	return vars
}

// checks the provided environment variables looking for feature flag based variables
func CheckFeatureFlag(key string, envVariables []lagoon.EnvironmentVariable, debug bool) string {
	// check for force value
	if value, ok := os.LookupEnv(fmt.Sprintf("LAGOON_FEATURE_FLAG_FORCE_%s", key)); ok {
		if debug {
			fmt.Println(fmt.Sprintf("Using forced flag value from build variable %s", fmt.Sprintf("LAGOON_FEATURE_FLAG_FORCE_%s", key)))
		}
		return value
	}
	// check lagoon environment variables
	for _, lVar := range envVariables {
		if strings.Contains(lVar.Name, fmt.Sprintf("LAGOON_FEATURE_FLAG_%s", key)) {
			if debug {
				fmt.Println(fmt.Sprintf("Using flag value from Lagoon environment variable %s", fmt.Sprintf("LAGOON_FEATURE_FLAG_%s", key)))
			}
			return lVar.Value
		}
	}
	// return default
	if value, ok := os.LookupEnv(fmt.Sprintf("LAGOON_FEATURE_FLAG_DEFAULT_%s", key)); ok {
		if debug {
			fmt.Println(fmt.Sprintf("Using default flag value from build variable %s", fmt.Sprintf("LAGOON_FEATURE_FLAG_DEFAULT_%s", key)))
		}
		return value
	}
	// otherwise nothing
	return ""
}

func CheckAdminFeatureFlag(key string, debug bool) string {
	if value, ok := os.LookupEnv(fmt.Sprintf("ADMIN_LAGOON_FEATURE_FLAG_%s", key)); ok {
		if debug {
			fmt.Println(fmt.Sprintf("Using admin feature flag value from build variable %s", fmt.Sprintf("ADMIN_LAGOON_FEATURE_FLAG_%s", key)))
		}
		return value
	}
	return ""
}

func ValidateResourceQuantity(s string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New(fmt.Sprint(x))
			}
		}
	}()
	resource.MustParse(s)
	return nil
}
