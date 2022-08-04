package generator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uselagoon/build-deploy-tool/internal/helpers"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
)

func generateRoutes(
	lagoonEnvVars []lagoon.EnvironmentVariable,
	buildValues BuildValues,
	lYAML lagoon.YAML,
	autogenRoutes *lagoon.RoutesV2,
	mainRoutes *lagoon.RoutesV2,
	activeStanbyRoutes *lagoon.RoutesV2,
	debug bool,
) (string, []string, []string, error) {
	var err error
	primary := ""
	remainders := []string{}
	autogen := []string{}
	prefix := "https://"

	// generate the autogenerated routes
	err = generateAutogenRoutes(lagoonEnvVars, &lYAML, &buildValues, autogenRoutes)
	if err != nil {
		return "", []string{}, []string{}, fmt.Errorf("couldn't unmarshal routes from Lagoon API, is it actually JSON that has been base64 encoded?: %v", err)
	}
	// get the first route from the list of routes
	if len(autogenRoutes.Routes) > 0 {
		for i := 0; i < len(autogenRoutes.Routes); i++ {
			autogen = append(autogen, fmt.Sprintf("%s%s", prefix, autogenRoutes.Routes[i].Domain))
			if i == 0 {
				primary = fmt.Sprintf("%s%s", prefix, autogenRoutes.Routes[i].Domain)
				// } else {
				// 	remainders = append(remainders, fmt.Sprintf("%s%s", prefix, autogenRoutes.Routes[i].Domain))
			}
			remainders = append(remainders, fmt.Sprintf("%s%s", prefix, autogenRoutes.Routes[i].Domain))
			for a := 0; a < len(autogenRoutes.Routes[i].AlternativeNames); a++ {
				remainders = append(remainders, fmt.Sprintf("%s%s", prefix, autogenRoutes.Routes[i].AlternativeNames[a]))
				autogen = append(autogen, fmt.Sprintf("%s%s", prefix, autogenRoutes.Routes[i].AlternativeNames[a]))
			}
		}
	}

	// handle routes from the .lagoon.yml and the API specifically
	err = generateIngress(lagoonEnvVars, buildValues, lYAML, mainRoutes, debug)
	if err != nil {
		return "", []string{}, []string{}, fmt.Errorf("couldn't generate and merge routes: %v", err)
	}

	// get the first route from the list of routes, replace the previous one if necessary
	if len(mainRoutes.Routes) > 0 {
		// if primary != "" {
		// 	remainders = append(remainders, primary)
		// }
		for i := 0; i < len(mainRoutes.Routes); i++ {
			if i == 0 {
				primary = fmt.Sprintf("%s%s", prefix, mainRoutes.Routes[i].Domain)
				// } else {
				// 	remainders = append(remainders, fmt.Sprintf("%s%s", prefix, mainRoutes.Routes[i].Domain))
			}
			remainders = append(remainders, fmt.Sprintf("%s%s", prefix, mainRoutes.Routes[i].Domain))
			for a := 0; a < len(mainRoutes.Routes[i].AlternativeNames); a++ {
				remainders = append(remainders, fmt.Sprintf("%s%s", prefix, mainRoutes.Routes[i].AlternativeNames[a]))
			}
		}
	}

	if buildValues.IsActiveEnvironment || buildValues.IsStandbyEnvironment {
		// active/standby routes should not be changed by any environment defined routes.
		// generate the templates for these independently of any previously generated routes,
		// this WILL overwrite previously created templates ensuring that anything defined in the `production_routes`
		// section are created correctly ensuring active/standby will work
		*activeStanbyRoutes = generateActiveStandbyRoutes(lagoonEnvVars, lYAML, buildValues)
		// get the first route from the list of routes, replace the previous one if necessary
		if len(activeStanbyRoutes.Routes) > 0 {
			// if primary != "" {
			// 	remainders = append(remainders, primary)
			// }
			for i := 0; i < len(activeStanbyRoutes.Routes); i++ {
				if i == 0 {
					primary = fmt.Sprintf("%s%s", prefix, activeStanbyRoutes.Routes[i].Domain)
					// } else {
					// 	remainders = append(remainders, fmt.Sprintf("%s%s", prefix, activeStanbyRoutes.Routes[i].Domain))
				}
				remainders = append(remainders, fmt.Sprintf("%s%s", prefix, activeStanbyRoutes.Routes[i].Domain))
				// remainders = append(remainders, fmt.Sprintf("%s%s", prefix, activeStanbyRoutes.Routes[i].Domain))
				for a := 0; a < len(activeStanbyRoutes.Routes[i].AlternativeNames); a++ {
					remainders = append(remainders, fmt.Sprintf("%s%s", prefix, activeStanbyRoutes.Routes[i].AlternativeNames[a]))
				}
			}
		}
	}

	return primary, remainders, autogen, nil
}

func generateIngress(
	envVars []lagoon.EnvironmentVariable,
	values BuildValues,
	lYAML lagoon.YAML,
	mainRoutes *lagoon.RoutesV2,
	debug bool,
) error {
	// read the routes from the API
	apiRoutes, err := getRoutesFromAPIEnvVar(envVars, debug)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal routes from Lagoon API, is it actually JSON that has been base64 encoded?: %v", err)
	}

	// handle routes from the .lagoon.yml and the API specifically
	*mainRoutes, err = generateAndMerge(*apiRoutes, envVars, lYAML, values)
	if err != nil {
		return fmt.Errorf("couldn't generate and merge routes: %v", err)
	}
	return nil
}

func generateAutogenRoutes(
	envVars []lagoon.EnvironmentVariable,
	lagoonYAML *lagoon.YAML,
	buildValues *BuildValues,
	autogenRoutes *lagoon.RoutesV2,
) error {
	// generate autogenerated routes for the services
	// get the router pattern
	lagoonRouterPattern, err := lagoon.GetLagoonVariable("LAGOON_SYSTEM_ROUTER_PATTERN", []string{"internal_system"}, envVars)
	if err == nil {
		fmt.Println(buildValues.Services)
		// if the `LAGOON_SYSTEM_ROUTER_PATTERN` exists, generate the routes
		for serviceName, service := range buildValues.Services {
			fmt.Println(serviceName)
			// get the service type
			// if autogenerated routes are enabled, generate them :)
			if service.AutogeneratedRoutesEnabled {
				// use the service name as the servicetype name
				serviceOverrideName := serviceName
				if service.OverrideName != "" {
					// but if a typename is provided by the service, use it instead
					serviceOverrideName = service.OverrideName
				}
				domain, shortDomain := autogeneratedDomainFromPattern(lagoonRouterPattern.Value, serviceOverrideName, buildValues.Project, buildValues.Environment)
				serviceValues := ServiceValues{
					AutogeneratedRouteDomain:      domain,
					ShortAutogeneratedRouteDomain: shortDomain,
				}
				buildValues.Services[serviceName] = serviceValues

				// alternativeNames are `prefixes` for autogenerated routes
				autgenPrefixes := lagoonYAML.Routes.Autogenerate.Prefixes
				alternativeNames := []string{}
				for _, altName := range autgenPrefixes {
					// add the prefix to the domain into a new slice of alternative domains
					alternativeNames = append(alternativeNames, fmt.Sprintf("%s.%s", altName, domain))
				}
				fastlyConfig := &lagoon.Fastly{}
				err := lagoon.GenerateFastlyConfiguration(fastlyConfig, buildValues.FastlyCacheNoCache, buildValues.Fastly.ServiceID, domain, buildValues.FastlyAPISecretPrefix, envVars)
				if err != nil {
					return err
				}
				insecure := "Allow"
				if lagoonYAML.Routes.Autogenerate.Insecure != "" {
					insecure = lagoonYAML.Routes.Autogenerate.Insecure
				}
				ingressClass := buildValues.IngressClass
				if lagoonYAML.Routes.Autogenerate.IngressClass != "" {
					ingressClass = lagoonYAML.Routes.Autogenerate.IngressClass
				}
				autogenRoute := lagoon.RouteV2{
					Domain:  domain,
					Fastly:  *fastlyConfig,
					TLSAcme: helpers.BoolPtr(service.AutogeneratedRoutesTLSAcme),
					// overwrite the custom-ingress labels
					Labels: map[string]string{
						"lagoon.sh/autogenerated":    "true",
						"helm.sh/chart":              fmt.Sprintf("%s-%s", "autogenerated-ingress", "0.1.0"),
						"app.kubernetes.io/name":     "autogenerated-ingress",
						"app.kubernetes.io/instance": serviceOverrideName,
						"lagoon.sh/service":          serviceOverrideName,
						"lagoon.sh/service-type":     service.Type,
					},
					IngressClass:     ingressClass,
					Autogenerated:    true,
					LagoonService:    serviceOverrideName,
					ComposeService:   serviceName,
					IngressName:      serviceOverrideName,
					Insecure:         &insecure,
					AlternativeNames: alternativeNames,
				}
				autogenRoutes.Routes = append(autogenRoutes.Routes, autogenRoute)
			}
		}
		return nil
	}
	// if there is no LAGOON_SYSTEM_ROUTER_PATTERN found, abort
	return err
}

// autogeneratedDomainFromPattern generates the domain name and the shortened domain name for an autogenerated ingress
func autogeneratedDomainFromPattern(pattern, service, projectName, environmentName string) (string, string) {
	domain := pattern
	shortDomain := pattern

	// fallback check for ${service} in the router pattern
	hasServicePattern := false
	if strings.Contains(pattern, "${service}") {
		hasServicePattern = true
	}

	// find and replace
	domain = strings.Replace(domain, "${service}", service, 1)
	domain = strings.Replace(domain, "${project}", projectName, 1)
	domain = strings.Replace(domain, "${environment}", environmentName, 1)
	// find and replace for the short domain
	shortDomain = strings.Replace(shortDomain, "${service}", service, 1)
	shortDomain = strings.Replace(shortDomain, "${project}", helpers.GetBase32EncodedLowercase(helpers.GetSha256Hash(projectName))[:8], 1)
	shortDomain = strings.Replace(shortDomain, "${environment}", helpers.GetBase32EncodedLowercase(helpers.GetSha256Hash(environmentName))[:8], 1)

	if !hasServicePattern {
		domain = fmt.Sprintf("%s.%s", service, domain)
		shortDomain = fmt.Sprintf("%s.%s", service, shortDomain)
	}

	domainParts := strings.Split(domain, ".")
	domainHash := helpers.GetSha256Hash(domain)
	finalDomain := ""
	for count, part := range domainParts {
		domainPart := part
		if len(part) > 63 {
			domainPart = fmt.Sprintf("%s-%s", part[:54], domainHash[:8])
		}
		if count == (len(domainParts) - 1) {
			finalDomain = fmt.Sprintf("%s%s", finalDomain, domainPart)
		} else {
			finalDomain = fmt.Sprintf("%s%s.", finalDomain, domainPart)
		}
	}
	return finalDomain, shortDomain
}

// create the activestandby routes from lagoon yaml
func generateActiveStandbyRoutes(
	envVars []lagoon.EnvironmentVariable,
	lagoonYAML lagoon.YAML,
	buildValues BuildValues,
) lagoon.RoutesV2 {
	activeStanbyRoutes := &lagoon.RoutesV2{}
	if lagoonYAML.ProductionRoutes != nil {
		if buildValues.IsActiveEnvironment == true {
			if lagoonYAML.ProductionRoutes.Active != nil {
				if lagoonYAML.ProductionRoutes.Active.Routes != nil {
					for _, routeMap := range lagoonYAML.ProductionRoutes.Active.Routes {
						lagoon.GenerateRoutesV2(activeStanbyRoutes, routeMap, envVars, buildValues.IngressClass, buildValues.FastlyAPISecretPrefix, true)
					}
				}
			}
		}
		if buildValues.IsStandbyEnvironment == true {
			if lagoonYAML.ProductionRoutes.Standby != nil {
				if lagoonYAML.ProductionRoutes.Standby.Routes != nil {
					for _, routeMap := range lagoonYAML.ProductionRoutes.Standby.Routes {
						lagoon.GenerateRoutesV2(activeStanbyRoutes, routeMap, envVars, buildValues.IngressClass, buildValues.FastlyAPISecretPrefix, true)
					}
				}
			}
		}
	}
	return *activeStanbyRoutes
}

// getRoutesFromEnvVar will collect the value of the LAGOON_ROUTES_JSON
// from provided lagoon environment variables from the API
func getRoutesFromAPIEnvVar(
	envVars []lagoon.EnvironmentVariable,
	debug bool,
) (*lagoon.RoutesV2, error) {
	apiRoutes := &lagoon.RoutesV2{}
	lagoonRoutesJSON, _ := lagoon.GetLagoonVariable("LAGOON_ROUTES_JSON", []string{"build", "global"}, envVars)
	if lagoonRoutesJSON != nil {
		if debug {
			fmt.Println("Collecting routes from environment variable LAGOON_ROUTES_JSON")
		}
		// if the routesJSON is populated, then attempt to decode and unmarshal it
		rawJSONStr, _ := base64.StdEncoding.DecodeString(lagoonRoutesJSON.Value)
		rawJSON := []byte(rawJSONStr)
		err := json.Unmarshal(rawJSON, apiRoutes)
		if err != nil {
			return nil, fmt.Errorf("couldn't unmarshal routes from Lagoon API, is it actually JSON that has been base64 encoded?: %v", err)
		}
	}
	return apiRoutes, nil
}

// generateAndMerge generates the completed custom ingress for an environment
// it generates the custom ingress from lagoon yaml and also merges in any that were
// provided by the lagoon environment variables from the API
func generateAndMerge(
	api lagoon.RoutesV2,
	envVars []lagoon.EnvironmentVariable,
	lagoonYAML lagoon.YAML,
	buildValues BuildValues,
) (lagoon.RoutesV2, error) {
	n := &lagoon.RoutesV2{} // placeholder for generated routes

	// otherwise it just uses the default environment name
	for _, routeMap := range lagoonYAML.Environments[buildValues.Branch].Routes {
		lagoon.GenerateRoutesV2(n, routeMap, envVars, buildValues.IngressClass, buildValues.FastlyAPISecretPrefix, false)
	}
	// merge routes from the API on top of the routes from the `.lagoon.yml`
	mainRoutes := lagoon.MergeRoutesV2(*n, api, envVars, buildValues.IngressClass, buildValues.FastlyAPISecretPrefix)
	return mainRoutes, nil
}
