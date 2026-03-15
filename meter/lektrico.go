package meter

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// lektricoMeterInfo maps the JSON response from meter_info.get
type lektricoMeterInfo struct {
	TotalActivePower    float64    `json:"total_active_power"`
	TotalImportedEnergy float64    `json:"total_imported_energy"`
	TotalExportedEnergy float64    `json:"total_exported_energy"`
	ActiveP             [3]float64 `json:"active_p"`
	Current             [3]float64 `json:"current"`
	Voltage             [3]float64 `json:"voltage"`
}

// LektricoMeter implements api.Meter for Lektrico M2W energy meters
type LektricoMeter struct {
	*request.Helper
	statusG util.Cacheable[lektricoMeterInfo]
}

var _ api.Meter = (*LektricoMeter)(nil)

func init() {
	registry.Add("lektrico", NewLektricoMeterFromConfig)
}

// NewLektricoMeterFromConfig creates a Lektrico meter from generic config
func NewLektricoMeterFromConfig(other map[string]any) (api.Meter, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	cc := struct {
		Host  string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.Host == "" {
		return nil, fmt.Errorf("missing host")
	}

	return NewLektricoMeter(cc.Host, cc.Cache)
}

// NewLektricoMeter creates a Lektrico meter
func NewLektricoMeter(host string, cache time.Duration) (*LektricoMeter, error) {
	log := util.NewLogger("lektrico")
	uri := fmt.Sprintf("http://%s/rpc", strings.TrimSuffix(host, "/"))

	wb := &LektricoMeter{
		Helper: request.NewHelper(log),
	}

	wb.statusG = util.ResettableCached(func() (lektricoMeterInfo, error) {
		var res lektricoMeterInfo
		err := wb.GetJSON(uri+"/meter_info.get", &res)
		return res, err
	}, cache)

	return wb, nil
}

// CurrentPower implements the api.Meter interface
func (wb *LektricoMeter) CurrentPower() (float64, error) {
	res, err := wb.statusG.Get()
	return res.TotalActivePower, err
}

var _ api.MeterEnergy = (*LektricoMeter)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *LektricoMeter) TotalEnergy() (float64, error) {
	res, err := wb.statusG.Get()
	return res.TotalImportedEnergy / 1000.0, err
}

var _ api.PhaseCurrents = (*LektricoMeter)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *LektricoMeter) Currents() (float64, float64, float64, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.Current[0], res.Current[1], res.Current[2], nil
}

var _ api.PhaseVoltages = (*LektricoMeter)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *LektricoMeter) Voltages() (float64, float64, float64, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.Voltage[0], res.Voltage[1], res.Voltage[2], nil
}

var _ api.PhasePowers = (*LektricoMeter)(nil)

// Powers implements the api.PhasePowers interface
func (wb *LektricoMeter) Powers() (float64, float64, float64, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.ActiveP[0], res.ActiveP[1], res.ActiveP[2], nil
}
