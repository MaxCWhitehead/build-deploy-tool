package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/uselagoon/build-deploy-tool/internal/dbaasclient"
	"github.com/uselagoon/build-deploy-tool/internal/helpers"
)

func TestIdentifyNativeCronjobs(t *testing.T) {
	type args struct {
		alertContact       string
		statusPageID       string
		projectName        string
		environmentName    string
		branch             string
		prNumber           string
		prHeadBranch       string
		prBaseBranch       string
		environmentType    string
		buildType          string
		activeEnvironment  string
		standbyEnvironment string
		cacheNoCache       string
		serviceID          string
		secretPrefix       string
		ingressClass       string
		rootlessWorkloads  string
		projectVars        string
		envVars            string
		lagoonVersion      string
		lagoonYAML         string
		valuesFilePath     string
		templatePath       string
		imageReferences    map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1 basic deployment",
			args: args{
				alertContact:    "alertcontact",
				statusPageID:    "statuspageid",
				projectName:     "example-project",
				environmentName: "main",
				environmentType: "production",
				buildType:       "branch",
				lagoonVersion:   "v2.7.x",
				branch:          "main",
				projectVars:     `[{"name":"LAGOON_SYSTEM_ROUTER_PATTERN","value":"${service}-${project}-${environment}.example.com","scope":"internal_system"}]`,
				envVars:         `[]`,
				lagoonYAML:      "../test-resources/template-lagoon-services/test1/lagoon.yml",
				templatePath:    "../test-resources/template-lagoon-services/output",
				imageReferences: map[string]string{
					"node": "harbor.example/example-project/main/node:latest",
				},
			},
			want: "[]",
		},
		{
			name: "test2a nginx-php deployment",
			args: args{
				alertContact:    "alertcontact",
				statusPageID:    "statuspageid",
				projectName:     "example-project",
				environmentName: "main",
				environmentType: "production",
				buildType:       "branch",
				lagoonVersion:   "v2.7.x",
				branch:          "main",
				projectVars:     `[{"name":"LAGOON_SYSTEM_ROUTER_PATTERN","value":"${service}-${project}-${environment}.example.com","scope":"internal_system"}]`,
				envVars:         `[]`,
				lagoonYAML:      "../test-resources/template-lagoon-services/test2/lagoon.yml",
				templatePath:    "../test-resources/template-lagoon-services/output",
				imageReferences: map[string]string{
					"nginx":   "harbor.example/example-project/main/nginx:latest",
					"php":     "harbor.example/example-project/main/php:latest",
					"cli":     "harbor.example/example-project/main/cli:latest",
					"redis":   "harbor.example/example-project/main/redis:latest",
					"varnish": "harbor.example/example-project/main/varnish:latest",
				},
			},
			want: `["cronjob-cli-drush-cron2"]`,
		},
		{
			name: "test2b nginx-php deployment - rootless",
			args: args{
				alertContact:      "alertcontact",
				statusPageID:      "statuspageid",
				projectName:       "example-project",
				environmentName:   "main",
				environmentType:   "production",
				buildType:         "branch",
				lagoonVersion:     "v2.7.x",
				branch:            "main",
				projectVars:       `[{"name":"LAGOON_SYSTEM_ROUTER_PATTERN","value":"${service}-${project}-${environment}.example.com","scope":"internal_system"}]`,
				envVars:           `[]`,
				rootlessWorkloads: "enabled",
				lagoonYAML:        "../test-resources/template-lagoon-services/test2/lagoon.yml",
				templatePath:      "../test-resources/template-lagoon-services/output",
				imageReferences: map[string]string{
					"nginx":   "harbor.example/example-project/main/nginx:latest",
					"php":     "harbor.example/example-project/main/php:latest",
					"cli":     "harbor.example/example-project/main/cli:latest",
					"redis":   "harbor.example/example-project/main/redis:latest",
					"varnish": "harbor.example/example-project/main/varnish:latest",
				},
			},
			want: `["cronjob-cli-drush-cron2"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set the environment variables from args
			err := os.Setenv("MONITORING_ALERTCONTACT", tt.args.alertContact)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("MONITORING_STATUSPAGEID", tt.args.statusPageID)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("PROJECT", tt.args.projectName)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("ENVIRONMENT", tt.args.environmentName)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("BRANCH", tt.args.branch)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("PR_NUMBER", tt.args.prNumber)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("PR_HEAD_BRANCH", tt.args.prHeadBranch)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("PR_BASE_BRANCH", tt.args.prBaseBranch)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("ENVIRONMENT_TYPE", tt.args.environmentType)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("BUILD_TYPE", tt.args.buildType)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("ACTIVE_ENVIRONMENT", tt.args.activeEnvironment)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("STANDBY_ENVIRONMENT", tt.args.standbyEnvironment)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("LAGOON_FASTLY_NOCACHE_SERVICE_ID", tt.args.cacheNoCache)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("LAGOON_PROJECT_VARIABLES", tt.args.projectVars)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("LAGOON_ENVIRONMENT_VARIABLES", tt.args.envVars)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("LAGOON_VERSION", tt.args.lagoonVersion)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("LAGOON_FEATURE_FLAG_DEFAULT_INGRESS_CLASS", tt.args.ingressClass)
			if err != nil {
				t.Errorf("%v", err)
			}
			err = os.Setenv("LAGOON_FEATURE_FLAG_DEFAULT_ROOTLESS_WORKLOAD", tt.args.rootlessWorkloads)
			if err != nil {
				t.Errorf("%v", err)
			}
			generator, err := generatorInput(false)
			if err != nil {
				t.Errorf("%v", err)
			}
			generator.LagoonYAML = tt.args.lagoonYAML
			generator.ImageReferences = tt.args.imageReferences
			generator.SavedTemplatesPath = tt.args.templatePath
			// add dbaasclient overrides for tests
			generator.DBaaSClient = dbaasclient.NewClient(dbaasclient.Client{
				RetryMax:     5,
				RetryWaitMin: time.Duration(10) * time.Millisecond,
				RetryWaitMax: time.Duration(50) * time.Millisecond,
			})

			savedTemplates := tt.args.templatePath
			err = os.MkdirAll(tt.args.templatePath, 0755)
			if err != nil {
				t.Errorf("couldn't create directory %v: %v", savedTemplates, err)
			}

			defer os.RemoveAll(savedTemplates)

			ts := dbaasclient.TestDBaaSHTTPServer()
			defer ts.Close()
			err = os.Setenv("DBAAS_OPERATOR_HTTP", ts.URL)
			if err != nil {
				t.Errorf("%v", err)
			}

			got, err := IdentifyNativeCronjobs(generator)
			if err != nil {
				t.Errorf("%v", err)
			}

			if got != tt.want {
				t.Errorf("IdentifyNativeCronjobs() = %v, want %v", got, tt.want)
			}

			t.Cleanup(func() {
				helpers.UnsetEnvVars([]helpers.EnvironmentVariable{{Name: "LAGOON_FEATURE_FLAG_DEFAULT_INGRESS_CLASS"}, {Name: "LAGOON_FEATURE_FLAG_ROOTLESS_WORKLOAD"}})
			})
		})
	}
}
