package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
)

type ingressIdentifyJSON struct {
	Primary       string   `json:"primary"`
	Secondary     []string `json:"secondary"`
	Autogenerated []string `json:"autogenerated"`
}

var primaryIngressIdentify = &cobra.Command{
	Use:     "primary-ingress",
	Aliases: []string{"pi"},
	Short:   "Identify the primary ingress for a specific environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		primary, _, _, err := IdentifyPrimaryIngress(false)
		if err != nil {
			return err
		}
		fmt.Println(primary)
		return nil
	},
}

var ingressIdentify = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"i"},
	Short:   "Identify all ingress for a specific environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		primary, secondary, autogen, err := IdentifyPrimaryIngress(false)
		if err != nil {
			return err
		}
		ret := ingressIdentifyJSON{
			Primary:       primary,
			Secondary:     secondary,
			Autogenerated: autogen,
		}
		retJSON, _ := json.Marshal(ret)
		fmt.Println(string(retJSON))
		return nil
	},
}

// IdentifyPrimaryIngress .
func IdentifyPrimaryIngress(debug bool) (string, []string, []string, error) {
	activeEnv := false
	standbyEnv := false

	lagoonEnvVars := []lagoon.EnvironmentVariable{}
	lagoonValues := lagoon.BuildValues{}
	lYAML := lagoon.YAML{}
	autogenRoutes := &lagoon.RoutesV2{}
	mainRoutes := &lagoon.RoutesV2{}
	activeStanbyRoutes := &lagoon.RoutesV2{}
	err := collectBuildValues(debug, &activeEnv, &standbyEnv, &lagoonEnvVars, &lagoonValues, &lYAML, autogenRoutes, mainRoutes, activeStanbyRoutes, ignoreNonStringKeyErrors)
	if err != nil {
		return "", []string{}, []string{}, err
	}

	return lagoonValues.Route, lagoonValues.Routes, lagoonValues.AutogeneratedRoutes, nil

}

func generateRoutes(lagoonEnvVars []lagoon.EnvironmentVariable,
	lagoonValues lagoon.BuildValues,
	lYAML lagoon.YAML,
	autogenRoutes *lagoon.RoutesV2, mainRoutes *lagoon.RoutesV2, activeStanbyRoutes *lagoon.RoutesV2,
	activeEnv, standbyEnv, debug bool,
) (string, []string, []string, error) {
	var err error
	primary := ""
	remainders := []string{}
	autogen := []string{}
	prefix := "https://"

	// collect the autogenerated routes
	err = generateAutogenRoutes(lagoonEnvVars, &lYAML, &lagoonValues, autogenRoutes)
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
	err = generateIngress(lagoonValues, lYAML, lagoonEnvVars, mainRoutes, debug)
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

	if activeEnv || standbyEnv {
		// active/standby routes should not be changed by any environment defined routes.
		// generate the templates for these independently of any previously generated routes,
		// this WILL overwrite previously created templates ensuring that anything defined in the `production_routes`
		// section are created correctly ensuring active/standby will work
		*activeStanbyRoutes = generateActiveStandby(activeEnv, standbyEnv, lagoonEnvVars, lYAML)
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

func init() {
	identifyCmd.AddCommand(primaryIngressIdentify)
	identifyCmd.AddCommand(ingressIdentify)
}
