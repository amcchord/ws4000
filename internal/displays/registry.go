package displays

import (
	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
)

func RegisterAll(nav *engine.Navigator, svc *weather.Service, cfg config.Config) {
	all := []engine.Display{
		NewHazards(svc, cfg),
		NewCurrentWeather(svc, cfg),
		NewLatestObservations(svc, cfg),
		NewHourly(svc, cfg),
		NewHourlyGraph(svc, cfg),
		NewTravelForecast(svc, cfg),
		NewRegionalForecast(svc, cfg),
		NewLocalForecast(svc, cfg),
		NewExtendedForecast(svc, cfg),
		NewAlmanac(svc, cfg),
		NewSPCOutlook(svc, cfg),
		NewRadar(svc, cfg),
	}
	for _, d := range all {
		if enabled, ok := cfg.Displays[d.ID()]; ok {
			d.SetEnabled(enabled)
		}
		nav.Register(d)
	}
}
