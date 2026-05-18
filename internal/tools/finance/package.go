package finance

import (
	"github.com/ff3300/aleph-v2/internal/repository"
)

func ListTools(metaRepo *repository.MetadataRepository) []interface {
	Register(metaRepo *repository.MetadataRepository) error
} {
	return []interface {
		Register(metaRepo *repository.MetadataRepository) error
	}{
		NewProphetForecastTool(),
		NewOpenBBMarketDataTool(),
		NewSentimentAnalysisFinTool(),
	}
}
