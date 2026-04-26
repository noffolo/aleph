package osint

import (
	"github.com/ff3300/aleph-v2/internal/repository"
)

// ListTools returns all OSINT tools registered in this package.
func ListTools(broker *Shadowbroker) []interface {
	Register(metaRepo *repository.MetadataRepository) error
} {
	return []interface {
		Register(metaRepo *repository.MetadataRepository) error
	}{
		NewRegionDossierTool(broker),
		NewThreatLevelTool(broker),
		NewVesselTrackingTool(broker),
		NewFlightTrackingTool(broker),
		NewCorrelationAlertsTool(broker),
	}
}
