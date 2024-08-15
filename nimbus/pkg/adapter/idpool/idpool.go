package idpool

import (
	"slices"
)

const (
	PkgManagerExecution = "pkgMgrs"
)

var IdsSupportedByKubeArmor = []string{
	PkgManagerExecution,
}

func IsSupportedId(id, engine string) bool {
	switch engine {
	case "kubearmor":
		return slices.Contains(IdsSupportedByKubeArmor, id)
	default:
		return false
	}
}
