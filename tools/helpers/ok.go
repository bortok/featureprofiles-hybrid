package helpers

import (
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
)

type ExpectedBgpMetrics struct {
	State      gosnappi.Bgpv4MetricSessionStateEnum
	Advertised int32
	Received   int32
}

type ExpectedIsisMetrics struct {
	L1SessionsUp   int32
	L2SessionsUp   int32
	L1DatabaseSize int32
	L2DatabaseSize int32
}
type ExpectedPortMetrics struct {
	FramesRx int32
}

type ExpectedFlowMetrics struct {
	FramesRx     int64
	FramesRxRate float32
}

type ExpectedState struct {
	Port map[string]ExpectedPortMetrics
	Flow map[string]ExpectedFlowMetrics
	Bgp4 map[string]ExpectedBgpMetrics
	Bgp6 map[string]ExpectedBgpMetrics
	Isis map[string]ExpectedIsisMetrics
}

func NewExpectedState() ExpectedState {
	e := ExpectedState{
		Port: map[string]ExpectedPortMetrics{},
		Flow: map[string]ExpectedFlowMetrics{},
		Bgp4: map[string]ExpectedBgpMetrics{},
		Bgp6: map[string]ExpectedBgpMetrics{},
		Isis: map[string]ExpectedIsisMetrics{},
	}
	return e
}

func Bgp4SessionAsExpected(t *testing.T, otg *ondatra.OTG, c gosnappi.Config, expectedState ExpectedState) (bool, error) {
	dMetrics, err := GetBgpv4Metrics(t, otg, c)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		Bgpv4Metrics:  dMetrics,
	})

	expected := true
	for _, d := range dMetrics.Items() {
		expectedMetrics := expectedState.Bgp4[d.Name()]
		if d.SessionState() != expectedMetrics.State || d.RoutesAdvertised() != expectedMetrics.Advertised || d.RoutesReceived() != expectedMetrics.Received {
			expected = false
		}
	}

	return expected, nil
}

func AllBgp4SessionUp(t *testing.T, otg *ondatra.OTG, c gosnappi.Config, expectedState ExpectedState) (bool, error) {
	dMetrics, err := GetBgpv4Metrics(t, otg, c)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		Bgpv4Metrics:  dMetrics,
	})

	expected := true
	for _, d := range dMetrics.Items() {
		expectedMetrics := expectedState.Bgp4[d.Name()]
		if d.SessionState() != gosnappi.Bgpv4MetricSessionState.UP || d.RoutesAdvertised() != expectedMetrics.Advertised || d.RoutesReceived() != expectedMetrics.Received {
			expected = false
		}
	}

	return expected, nil
}

func AllBgp6SessionUp(t *testing.T, otg *ondatra.OTG, c gosnappi.Config, expectedState ExpectedState) (bool, error) {
	dMetrics, err := GetBgpv6Metrics(t, otg, c)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		Bgpv6Metrics:  dMetrics,
	})

	expected := true
	for _, d := range dMetrics.Items() {
		expectedMetrics := expectedState.Bgp6[d.Name()]
		if d.SessionState() != gosnappi.Bgpv6MetricSessionState.UP || d.RoutesAdvertised() != expectedMetrics.Advertised || d.RoutesReceived() != expectedMetrics.Received {
			expected = false
		}
	}

	return expected, nil
}

func AllIsisSessionUp(t *testing.T, otg *ondatra.OTG, c gosnappi.Config, expectedState ExpectedState) (bool, error) {
	dMetrics, err := GetIsisMetrics(t, otg, c)
	if err != nil {
		return false, err
	}
	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		IsisMetrics:   dMetrics,
	})
	expected := true
	for _, d := range dMetrics.Items() {
		expectedMetrics := expectedState.Isis[d.Name()]
		if d.L1SessionsUp() != expectedMetrics.L1SessionsUp || d.L1DatabaseSize() != expectedMetrics.L1DatabaseSize || d.L1SessionsUp() != expectedMetrics.L1SessionsUp || d.L2DatabaseSize() != expectedMetrics.L2DatabaseSize {
			expected = false
		}
	}

	// TODO: wait explicitly until telemetry API (for talking to DUT) is available
	if expected {
		time.Sleep(2 * time.Second)
	}
	return expected, nil
}

func FlowMetricsOk(t *testing.T, otg *ondatra.OTG, c gosnappi.Config, expectedState ExpectedState) (bool, error) {
	fMetrics, err := GetFlowMetrics(t, otg, c)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		FlowMetrics:   fMetrics,
	})

	expected := true
	for _, f := range fMetrics.Items() {
		expectedMetrics := expectedState.Flow[f.Name()]
		if f.FramesRx() != expectedMetrics.FramesRx || f.FramesRxRate() != expectedMetrics.FramesRxRate {
			expected = false
		}
	}

	return expected, nil
}

func PortAndFlowMetricsOk(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (bool, error) {
	expected := 0
	for _, f := range c.Flows().Items() {
		expected += int(f.Duration().FixedPackets().Packets())
	}

	fMetrics, err := GetFlowMetrics(t, otg, c)
	if err != nil {
		return false, err
	}

	pMetrics, err := GetPortMetrics(t, otg, c)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		FlowMetrics:   fMetrics,
		PortMetrics:   pMetrics,
	})

	actual := 0
	for _, m := range fMetrics.Items() {
		actual += int(m.FramesRx())
	}

	return expected == actual, nil
}

func ArpEntriesOk(t *testing.T, otg *ondatra.OTG, ipType string, expectedMacEntries []string) (bool, error) {
	actualMacEntries := []string{}
	var err error
	switch ipType {
	case "IPv4":
		actualMacEntries, err = GetAllIPv4NeighborMacEntries(t, otg)
		if err != nil {
			return false, err
		}
	case "IPv6":
		actualMacEntries, err = GetAllIPv6NeighborMacEntries(t, otg)
		if err != nil {
			return false, err
		}
	}

	t.Logf("Expected Mac Entries: %v", expectedMacEntries)
	t.Logf("OTG Mac Entries: %v", actualMacEntries)

	expected := true
	expected = expectedElementsPresent(expectedMacEntries, actualMacEntries)
	return expected, nil
}