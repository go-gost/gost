package metrics

var (
	nilGauge    = &nopGauge{}
	nilCounter  = &nopCounter{}
	nilObserver = &nopObserver{}
)

type nopGauge struct{}

func (*nopGauge) Inc()          {}
func (*nopGauge) Dec()          {}
func (*nopGauge) Add(v float64) {}
func (*nopGauge) Set(v float64) {}

type nopCounter struct{}

func (*nopCounter) Inc()          {}
func (*nopCounter) Add(v float64) {}

type nopObserver struct{}

func (*nopObserver) Observe(v float64) {}
