package services

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/uselagoon/build-deploy-tool/internal/generator"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
)

func TestGenerateDeploymentTemplate(t *testing.T) {
	type args struct {
		buildValues generator.BuildValues
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test1 - basic",
			args: args{
				buildValues: generator.BuildValues{
					Project:         "example-project",
					Environment:     "environment-name",
					EnvironmentType: "production",
					Namespace:       "myexample-project-environment-name",
					BuildType:       "branch",
					LagoonVersion:   "v2.x.x",
					Kubernetes:      "generator.local",
					Branch:          "environment-name",
					ImagePullSecrets: []generator.ImagePullSecrets{
						{
							Name: "lagoon-internal-registry-secret",
						},
					},
					GitSha:       "0",
					ConfigMapSha: "32bf1359ac92178c8909f0ef938257b477708aa0d78a5a15ad7c2d7919adf273",
					Services: []generator.ServiceValues{
						{
							Name:             "myservice",
							OverrideName:     "myservice",
							Type:             "basic",
							DBaaSEnvironment: "production",
							ImageName:        "harbor.example.com/example-project/environment-name/myservice@latest",
							InPodCronjobs: []lagoon.Cronjob{
								{
									Name:     "cron - inpod",
									Schedule: "M/5 * * * *",
									Command:  "drush cron",
									Service:  "basic",
								},
								{
									Name:     "cron2 - inpod",
									Schedule: "M/15 * * * *",
									Command:  "other cronjob",
									Service:  "basic",
								},
							},
							NativeCronjobs: []lagoon.Cronjob{
								{
									Name:     "cron3 - native",
									Schedule: "35 * * * *",
									Command:  "drush cron",
									Service:  "basic",
								},
							},
						},
						{
							Name:             "myservice-po",
							OverrideName:     "myservice-po",
							Type:             "basic",
							DBaaSEnvironment: "production",
							ImageName:        "harbor.example.com/example-project/environment-name/myservice-po@latest",
							ServicePort:      8080, // template should have port changed to 8080 from 3000
							UseSpotInstances: true, // template should have spot instance label and toleration/selector/affinity
							Replicas:         2,
						},
						// {
						// 	Name:                 "myservice-persist",
						// 	OverrideName:         "myservice-persist",
						// 	Type:                 "basic-persistent",
						// 	DBaaSEnvironment:     "production",
						// 	PersistentVolumeSize: "5Gi",
						// 	PersistentVolumePath: "/storage/data",
						// 	PersistentVolumeName: "basic",
						// },
						// {
						// 	Name:                 "myservice-persist-po",
						// 	OverrideName:         "myservice-persist-po",
						// 	Type:                 "basic-persistent",
						// 	DBaaSEnvironment:     "production",
						// 	ServicePort:          8080,
						// 	PersistentVolumeSize: "5Gi",
						// 	PersistentVolumePath: "/storage/data",
						// 	PersistentVolumeName: "basic",
						// },
						// {
						// 	Name:                 "myservice-persist-po",
						// 	OverrideName:         "myservice-persist-po",
						// 	Type:                 "basic-persistent",
						// 	DBaaSEnvironment:     "production",
						// 	ServicePort:          8080,
						// 	PersistentVolumeSize: "100Gi",
						// 	PersistentVolumePath: "/storage/data",
						// 	PersistentVolumeName: "basic",
						// },
					},
				},
			},
			want: "test-resources/deployment/result-basic-1.yaml",
		},
		{
			name: "test2 - nginx-php",
			args: args{
				buildValues: generator.BuildValues{
					Project:         "example-project",
					Environment:     "environment-name",
					EnvironmentType: "production",
					Namespace:       "myexample-project-environment-name",
					BuildType:       "branch",
					LagoonVersion:   "v2.x.x",
					Kubernetes:      "generator.local",
					Branch:          "environment-name",
					ImagePullSecrets: []generator.ImagePullSecrets{
						{
							Name: "lagoon-internal-registry-secret",
						},
					},
					GitSha:       "0",
					ConfigMapSha: "32bf1359ac92178c8909f0ef938257b477708aa0d78a5a15ad7c2d7919adf273",
					Services: []generator.ServiceValues{
						{
							Name:             "nginx",
							OverrideName:     "nginx",
							Type:             "nginx-php",
							DBaaSEnvironment: "production",
							ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
						},
						{
							Name:             "php",
							OverrideName:     "nginx",
							Type:             "nginx-php",
							DBaaSEnvironment: "production",
							ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
						},
						{
							Name:                 "nginx2",
							OverrideName:         "nginx2",
							Type:                 "nginx-php-persistent",
							DBaaSEnvironment:     "production",
							ImageName:            "harbor.example.com/example-project/environment-name/nginx2@latest",
							PersistentVolumeSize: "10Gi",
							PersistentVolumePath: "/storage/data",
							PersistentVolumeName: "nginx2",
						},
						{
							Name:                 "php2",
							OverrideName:         "nginx2",
							Type:                 "nginx-php-persistent",
							DBaaSEnvironment:     "production",
							ImageName:            "harbor.example.com/example-project/environment-name/php2@latest",
							PersistentVolumeSize: "10Gi",
							PersistentVolumePath: "/storage/data",
							PersistentVolumeName: "nginx2",
						},
					},
				},
			},
			want: "test-resources/deployment/result-nginx-1.yaml",
		},
		{
			name: "test2 - cli",
			args: args{
				buildValues: generator.BuildValues{
					Project:         "example-project",
					Environment:     "environment-name",
					EnvironmentType: "production",
					Namespace:       "myexample-project-environment-name",
					BuildType:       "branch",
					LagoonVersion:   "v2.x.x",
					Kubernetes:      "generator.local",
					Branch:          "environment-name",
					ImagePullSecrets: []generator.ImagePullSecrets{
						{
							Name: "lagoon-internal-registry-secret",
						},
					},
					GitSha:       "0",
					ConfigMapSha: "32bf1359ac92178c8909f0ef938257b477708aa0d78a5a15ad7c2d7919adf273",
					Services: []generator.ServiceValues{
						{
							Name:             "myservice",
							OverrideName:     "myservice",
							Type:             "cli",
							DBaaSEnvironment: "production",
							ImageName:        "harbor.example.com/example-project/environment-name/myservice@latest",
						},
						{
							Name:                 "myservice-persist",
							OverrideName:         "myservice-persist",
							Type:                 "cli-persistent",
							DBaaSEnvironment:     "production",
							ImageName:            "harbor.example.com/example-project/environment-name/myservice-persistent@latest",
							PersistentVolumeSize: "10Gi",
							PersistentVolumePath: "/storage/data",
							PersistentVolumeName: "myservice-persistent",
						},
					},
				},
			},
			want: "test-resources/deployment/result-cli-1.yaml",
		},
		// {
		// 	name: "test3 - elasticsearch",
		// 	args: args{
		// 		buildValues: generator.BuildValues{
		// 			Project:         "example-project",
		// 			Environment:     "environment-name",
		// 			EnvironmentType: "production",
		// 			Namespace:       "myexample-project-environment-name",
		// 			BuildType:       "branch",
		// 			LagoonVersion:   "v2.x.x",
		// 			Kubernetes:      "generator.local",
		// 			Branch:          "environment-name",
		// 			Services: []generator.ServiceValues{
		// 				{
		// 					Name:                 "myservice",
		// 					OverrideName:         "myservice",
		// 					Type:                 "elasticsearch",
		// 					DBaaSEnvironment:     "production",
		// 					PersistentVolumeSize: "5Gi",
		// 				},
		// 				{
		// 					Name:                 "myservice-size",
		// 					OverrideName:         "myservice-size",
		// 					Type:                 "elasticsearch",
		// 					DBaaSEnvironment:     "production",
		// 					PersistentVolumeSize: "100Gi",
		// 				},
		// 			},
		// 		},
		// 	},
		// 	want: "test-resources/deployment/result-elasticsearch-1.yaml",
		// },
		// {
		// 	name: "test4 - opensearch",
		// 	args: args{
		// 		buildValues: generator.BuildValues{
		// 			Project:         "example-project",
		// 			Environment:     "environment-name",
		// 			EnvironmentType: "production",
		// 			Namespace:       "myexample-project-environment-name",
		// 			BuildType:       "branch",
		// 			LagoonVersion:   "v2.x.x",
		// 			Kubernetes:      "generator.local",
		// 			Branch:          "environment-name",
		// 			Services: []generator.ServiceValues{
		// 				{
		// 					Name:                 "myservice",
		// 					OverrideName:         "myservice",
		// 					Type:                 "opensearch",
		// 					DBaaSEnvironment:     "production",
		// 					PersistentVolumeSize: "5Gi",
		// 				},
		// 				{
		// 					Name:                 "myservice-size",
		// 					OverrideName:         "myservice-size",
		// 					Type:                 "opensearch",
		// 					DBaaSEnvironment:     "production",
		// 					PersistentVolumeSize: "100Gi",
		// 				},
		// 			},
		// 		},
		// 	},
		// 	want: "test-resources/deployment/result-opensearch-1.yaml",
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateDeploymentTemplate(tt.args.buildValues)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateDeploymentTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			r1, err := os.ReadFile(tt.want)
			if err != nil {
				t.Errorf("couldn't read file %v: %v", tt.want, err)
			}
			if !reflect.DeepEqual(string(got), string(r1)) {
				t.Errorf("GenerateDeploymentTemplate() = \n%v", diff.LineDiff(string(r1), string(got)))
			}
		})
	}
}

func TestLinkedServiceCalculator(t *testing.T) {
	type args struct {
		services []generator.ServiceValues
	}
	tests := []struct {
		name string
		args args
		want []generator.ServiceValues
	}{
		{
			name: "test1 - standard nginx-php",
			args: args{
				services: []generator.ServiceValues{
					{
						Name:             "nginx",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
					},
					{
						Name:             "php",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
					},
				},
			},
			want: []generator.ServiceValues{
				{
					Name:             "nginx",
					OverrideName:     "nginx",
					Type:             "nginx-php",
					DBaaSEnvironment: "production",
					ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
					LinkedService: &generator.ServiceValues{
						Name:             "php",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
					},
				},
			},
		},
		{
			name: "test2 - multiple linked services (2 separate nginx-php)",
			args: args{
				services: []generator.ServiceValues{
					{
						Name:             "nginx",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
					},
					{
						Name:             "php",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
					},
					{
						Name:             "nginx2",
						OverrideName:     "nginx2",
						Type:             "nginx-php-persistent",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/nginx2@latest",
					},
					{
						Name:             "php2",
						OverrideName:     "nginx2",
						Type:             "nginx-php-persistent",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php2@latest",
					},
				},
			},
			want: []generator.ServiceValues{
				{
					Name:             "nginx",
					OverrideName:     "nginx",
					Type:             "nginx-php",
					DBaaSEnvironment: "production",
					ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
					LinkedService: &generator.ServiceValues{
						Name:             "php",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
					},
				},
				{
					Name:             "nginx2",
					OverrideName:     "nginx2",
					Type:             "nginx-php-persistent",
					DBaaSEnvironment: "production",
					ImageName:        "harbor.example.com/example-project/environment-name/nginx2@latest",
					LinkedService: &generator.ServiceValues{
						Name:             "php2",
						OverrideName:     "nginx2",
						Type:             "nginx-php-persistent",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php2@latest",
					},
				},
			},
		},
		{
			name: "test3 - single nginx-php and a single nginx standalone",
			args: args{
				services: []generator.ServiceValues{
					{
						Name:             "nginx",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
					},
					{
						Name:             "php",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
					},
					{
						Name:             "normalnginx",
						OverrideName:     "normalnginx",
						Type:             "nginx",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/normalnginx@latest",
					},
				},
			},
			want: []generator.ServiceValues{
				{
					Name:             "normalnginx",
					OverrideName:     "normalnginx",
					Type:             "nginx",
					DBaaSEnvironment: "production",
					ImageName:        "harbor.example.com/example-project/environment-name/normalnginx@latest",
				},
				{
					Name:             "nginx",
					OverrideName:     "nginx",
					Type:             "nginx-php",
					DBaaSEnvironment: "production",
					ImageName:        "harbor.example.com/example-project/environment-name/nginx@latest",
					LinkedService: &generator.ServiceValues{
						Name:             "php",
						OverrideName:     "nginx",
						Type:             "nginx-php",
						DBaaSEnvironment: "production",
						ImageName:        "harbor.example.com/example-project/environment-name/php@latest",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LinkedServiceCalculator(tt.args.services)
			lValues, _ := json.Marshal(got)
			wValues, _ := json.Marshal(tt.want)
			if !reflect.DeepEqual(string(lValues), string(wValues)) {
				t.Errorf("LinkedServiceCalculator() = %v, want %v", string(lValues), string(wValues))
			}
		})
	}
}
