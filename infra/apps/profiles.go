package apps

import (
	"github.com/pkg/errors"

	"github.com/CoreumFoundation/crust/infra"
	"github.com/CoreumFoundation/crust/infra/apps/bdjuno"
	"github.com/CoreumFoundation/crust/infra/apps/cored"
	"github.com/CoreumFoundation/crust/infra/apps/faucet"
	"github.com/CoreumFoundation/crust/infra/apps/gaiad"
	"github.com/CoreumFoundation/crust/infra/apps/hermes"
	"github.com/CoreumFoundation/crust/infra/apps/osmosis"
)

// TestGroup constant values.
const (
	TestGroupCoreumModules = "coreum-modules"
	TestGroupCoreumUpgrade = "coreum-upgrade"
	TestGroupCoreumIBC     = "coreum-ibc"
	TestGroupFaucet        = "faucet"
)

// AppPrefix constants are the prefixes used in the app factories.
const (
	AppPrefixCored      = "cored"
	AppPrefixIBC        = "ibc"
	AppPrefixExplorer   = "explorer"
	AppPrefixMonitoring = "monitoring"
	AppPrefixXRPL       = "xrpl"
)

// Predefined Profiles.
const (
	Profile1Cored                  = "1cored"
	Profile3Cored                  = "3cored"
	Profile5Cored                  = "5cored"
	ProfileDevNet                  = "devnet"
	ProfileIBC                     = "ibc"
	ProfileFaucet                  = "faucet"
	ProfileExplorer                = "explorer"
	ProfileMonitoring              = "monitoring"
	ProfileXRPL                    = "xrpl"
	ProfileIntegrationTestsIBC     = "integration-tests-ibc"
	ProfileIntegrationTestsModules = "integration-tests-modules"
)

var profiles = []string{
	Profile1Cored,
	Profile3Cored,
	Profile5Cored,
	ProfileDevNet,
	ProfileIBC,
	ProfileFaucet,
	ProfileExplorer,
	ProfileMonitoring,
	ProfileXRPL,
	ProfileIntegrationTestsIBC,
	ProfileIntegrationTestsModules,
}

var defaultProfiles = []string{Profile1Cored}

var availableProfiles = func() map[string]struct{} {
	v := map[string]struct{}{}
	for _, p := range profiles {
		v[p] = struct{}{}
	}
	return v
}()

// Profiles returns the list of available profiles.
func Profiles() []string {
	return profiles
}

// DefaultProfiles returns the list of default profiles started if user didn't provide anything else.
func DefaultProfiles() []string {
	return defaultProfiles
}

// BuildAppSet builds the application set to deploy based on provided profiles.
func BuildAppSet(appF *Factory, profiles []string, coredVersion string) (infra.AppSet, cored.Cored, error) {
	pMap, err := checkProfiles(profiles)
	if err != nil {
		return nil, cored.Cored{}, err
	}

	if pMap[ProfileIntegrationTestsIBC] || pMap[ProfileIntegrationTestsModules] {
		if pMap[Profile1Cored] {
			return nil, cored.Cored{}, errors.Errorf(
				"profile 1cored can't be used together with integration-tests as it requires 3cored, 5cored or devnet",
			)
		}
		if !pMap[Profile5Cored] && !pMap[ProfileDevNet] {
			pMap[Profile3Cored] = true
		}
	}

	if pMap[ProfileIntegrationTestsIBC] {
		pMap[ProfileIBC] = true
	}

	if (pMap[ProfileIBC] || pMap[ProfileFaucet] || pMap[ProfileExplorer] || pMap[ProfileMonitoring]) &&
		!pMap[Profile3Cored] && !pMap[Profile5Cored] && !pMap[ProfileDevNet] {
		pMap[Profile1Cored] = true
	}

	validatorCount, sentryCount, seedCount, fullCount := decideNumOfCoredNodes(pMap)

	var coredApp cored.Cored
	var appSet infra.AppSet

	coredApp, coredNodes, err := appF.CoredNetwork(
		AppPrefixCored,
		cored.DefaultPorts,
		validatorCount, sentryCount, seedCount, fullCount,
		coredVersion,
	)
	if err != nil {
		return nil, cored.Cored{}, err
	}
	for _, coredNode := range coredNodes {
		appSet = append(appSet, coredNode)
	}

	if pMap[ProfileIBC] {
		appSet = append(appSet, appF.IBC(AppPrefixIBC, coredApp)...)
	}

	var faucetApp faucet.Faucet
	if pMap[ProfileFaucet] {
		appSet = append(appSet, appF.Faucet(string(faucet.AppType), coredApp))
	}

	if pMap[ProfileExplorer] {
		appSet = append(appSet, appF.BlockExplorer(AppPrefixExplorer, coredApp).ToAppSet()...)
	}

	if pMap[ProfileMonitoring] {
		var bdJunoApp bdjuno.BDJuno
		if bdJunoAppSetApp, ok := appSet.FindAppByName(
			BuildPrefixedAppName(AppPrefixExplorer, string(bdjuno.AppType)),
		).(bdjuno.BDJuno); ok {
			bdJunoApp = bdJunoAppSetApp
		}

		var hermesApps []hermes.Hermes
		if hermesAppSetApp, ok := appSet.FindAppByName(
			BuildPrefixedAppName(AppPrefixIBC, string(hermes.AppType), string(gaiad.AppType)),
		).(hermes.Hermes); ok {
			hermesApps = append(hermesApps, hermesAppSetApp)
		}

		if hermesAppSetApp, ok := appSet.FindAppByName(
			BuildPrefixedAppName(AppPrefixIBC, string(hermes.AppType), string(osmosis.AppType)),
		).(hermes.Hermes); ok {
			hermesApps = append(hermesApps, hermesAppSetApp)
		}

		appSet = append(appSet, appF.Monitoring(
			AppPrefixMonitoring,
			coredNodes,
			faucetApp,
			bdJunoApp,
			hermesApps,
		)...)
	}

	if pMap[ProfileXRPL] {
		appSet = append(appSet, appF.XRPL(
			AppPrefixXRPL,
		))
	}

	return appSet, coredApp, nil
}

func decideNumOfCoredNodes(pMap map[string]bool) (validatorCount, sentryCount, seedCount, fullCount int) {
	switch {
	case pMap[Profile1Cored]:
		return 1, 0, 0, 0
	case pMap[Profile3Cored]:
		return 3, 0, 0, 0
	case pMap[Profile5Cored]:
		return 5, 0, 0, 0
	case pMap[ProfileDevNet]:
		return 3, 1, 1, 2
	default:
		panic("no cored profile specified.")
	}
}

func checkProfiles(profiles []string) (map[string]bool, error) {
	pMap := map[string]bool{}
	coredProfilePresent := false
	for _, p := range profiles {
		if _, ok := availableProfiles[p]; !ok {
			return nil, errors.Errorf("profile %s does not exist", p)
		}
		if p == Profile1Cored || p == Profile3Cored || p == Profile5Cored {
			if coredProfilePresent {
				return nil, errors.Errorf("profiles 1cored, 3cored and 5cored are mutually exclusive")
			}
			coredProfilePresent = true
		}
		pMap[p] = true
	}

	return pMap, nil
}
