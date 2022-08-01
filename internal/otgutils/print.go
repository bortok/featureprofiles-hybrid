package otgutils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra/otg"
	otgtelemetry "github.com/openconfig/ondatra/telemetry/otg"
	"github.com/openconfig/ygot/ygot"
)

// LogFlowMetrics is displaying the otg flow statistics.
func LogFlowMetrics(t testing.TB, otg *otg.OTG, c gosnappi.Config) {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nFlow Metrics\n")
	for i := 1; i <= 80; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out, "%-25v%-15v%-15v%-15v%-15v\n", "Name", "Frames Tx", "Frames Rx", "FPS Tx", "FPS Rx")
	for _, f := range c.Flows().Items() {
		flowMetrics := otg.Telemetry().Flow(f.Name()).Get(t)
		rxPkts := flowMetrics.GetCounters().GetInPkts()
		txPkts := flowMetrics.GetCounters().GetOutPkts()
		rxRate := ygot.BinaryToFloat32(flowMetrics.GetInFrameRate())
		txRate := ygot.BinaryToFloat32(flowMetrics.GetOutFrameRate())
		out.WriteString(fmt.Sprintf("%-25v%-15v%-15v%-15v%-15v\n", f.Name(), txPkts, rxPkts, txRate, rxRate))
	}
	fmt.Fprintln(&out, strings.Repeat("-", 80))
	out.WriteString("\n\n")
	t.Log(out.String())
}

// LogPortMetrics is displaying otg port stats.
func LogPortMetrics(t testing.TB, otg *otg.OTG, c gosnappi.Config) {
	t.Helper()
	var link string
	var out strings.Builder
	out.WriteString("\nPort Metrics\n")
	for i := 1; i <= 120; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out,
		"%-25s%-15s%-15s%-15s%-15s%-15s%-15s%-15s\n",
		"Name", "Frames Tx", "Frames Rx", "Bytes Tx", "Bytes Rx", "FPS Tx", "FPS Rx", "Link")
	for _, p := range c.Ports().Items() {
		portMetrics := otg.Telemetry().Port(p.Name()).Get(t)
		rxFrames := portMetrics.GetCounters().GetInFrames()
		txFrames := portMetrics.GetCounters().GetOutFrames()
		rxRate := ygot.BinaryToFloat32(portMetrics.GetInRate())
		txRate := ygot.BinaryToFloat32(portMetrics.GetOutRate())
		rxBytes := portMetrics.GetCounters().GetInOctets()
		txBytes := portMetrics.GetCounters().GetOutOctets()
		link = "down"
		if portMetrics.GetLink() == otgtelemetry.Port_Link_UP {
			link = "up"
		}
		out.WriteString(fmt.Sprintf(
			"%-25v%-15v%-15v%-15v%-15v%-15v%-15v%-15v\n",
			p.Name(), txFrames, rxFrames, txBytes, rxBytes, txRate, rxRate, link,
		))
	}
	fmt.Fprintln(&out, strings.Repeat("-", 120))
	out.WriteString("\n\n")
	t.Log(out.String())
}

// LogLagMetrics is displaying otg lag stats.
func LogLagMetrics(t testing.TB, otg *otg.OTG, c gosnappi.Config) {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nOTG LAG Metrics\n")
	for i := 1; i <= 120; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out,
		"%-25s%-15s%-20s\n",
		"Name", "Oper Status", "Member Ports UP")
	for _, lag := range c.Lags().Items() {
		lagMetrics := otg.Telemetry().Lag(lag.Name()).Get(t)
		operStatus := lagMetrics.GetOperStatus().String()
		memberPortsUP := lagMetrics.GetCounters().GetMemberPortsUp()
		// framesTx := lagMetrics.GetCounters().GetOutFrames()
		// framesRx := lagMetrics.GetCounters().GetInFrames()
		out.WriteString(fmt.Sprintf(
			"%-25v%-15v%-20v\n",
			lag.Name(), operStatus, memberPortsUP,
		))
	}
	fmt.Fprintln(&out, strings.Repeat("-", 120))
	out.WriteString("\n\n")
	t.Log(out.String())
}

// LogLacpMetrics is displaying otg lacp stats.
func LogLacpMetrics(t testing.TB, otg *otg.OTG, c gosnappi.Config) {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nOTG LACP Metrics\n")
	for i := 1; i <= 120; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out,
		"%-10s%-15s%-18s%-15s%-15s%-20s%-20s\n",
		"LAG",
		"Member Port",
		"Synchronization",
		"Collecting",
		"Distributing",
		"System Id",
		"Partner Id")

	for _, lag := range c.Lags().Items() {
		lagPorts := lag.Ports().Items()
		for _, lagPort := range lagPorts {
			lacpMetric := otg.Telemetry().Lacp().LagMember(lagPort.PortName()).Get(t)
			synchronization := lacpMetric.GetSynchronization().String()
			collecting := lacpMetric.GetCollecting()
			distributing := lacpMetric.GetDistributing()
			systemId := lacpMetric.GetSystemId()
			partnerId := lacpMetric.GetPartnerId()
			out.WriteString(fmt.Sprintf(
				"%-10v%-15v%-18v%-15v%-15v%-20v%-20v\n",
				lag.Name(), lagPort.PortName(), synchronization, collecting, distributing, systemId, partnerId,
			))

		}
	}
	fmt.Fprintln(&out, strings.Repeat("-", 120))
	out.WriteString("\n\n")
	t.Log(out.String())
}

// LogBGPv4Metrics is displaying otg BGPv4 stats.
func LogBGPv4Metrics(t testing.TB, otg *otg.OTG, c gosnappi.Config) {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nOTG BGPv4 Metrics\n")
	for i := 1; i <= 140; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out,
		"%-15s%-15s%-15s%-15s%-15s%-20s%-20s%-20s%-15s\n",
		"Name",
		"State",
		"Flaps",
		"Routes TX",
		"Routes RX",
		"Route Withdraws Tx",
		"Route Withdraws Rx",
		"Keepalives Tx",
		"Keepalives Rx",
	)

	for _, d := range c.Devices().Items() {
		for _, ip := range d.Bgp().Ipv4Interfaces().Items() {
			for _, peer := range ip.Peers().Items() {
				bgpv4Metric := otg.Telemetry().BgpPeer(peer.Name()).Get(t)
				name := bgpv4Metric.GetName()
				state := bgpv4Metric.GetSessionState().String()
				flaps := bgpv4Metric.GetCounters().GetFlaps()
				routesTx := bgpv4Metric.GetCounters().GetOutRoutes()
				routesRx := bgpv4Metric.GetCounters().GetInRoutes()
				routesWithdrawTx := bgpv4Metric.GetCounters().GetOutRouteWithdraw()
				routesWithdrawRx := bgpv4Metric.GetCounters().GetInRouteWithdraw()
				keepalivesTx := bgpv4Metric.GetCounters().GetOutKeepalives()
				keepalivesRx := bgpv4Metric.GetCounters().GetInKeepalives()
				out.WriteString(fmt.Sprintf(
					"%-15v%-15v%-15v%-15v%-15v%-20v%-20v%-20v%-15v\n",
					name, state, flaps, routesTx, routesRx, routesWithdrawTx, routesWithdrawRx, keepalivesTx, keepalivesRx,
				))

			}
		}
	}
	fmt.Fprintln(&out, strings.Repeat("-", 140))
	out.WriteString("\n\n")
	t.Log(out.String())
}

// LogISISMetrics is displaying otg ISIS stats.
func LogISISMetrics(t testing.TB, otg *otg.OTG, c gosnappi.Config) {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nOTG ISIS Metrics\n")
	for i := 1; i <= 120; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out,
		"%-15s%-15s%-15s%-15s%-15s\n",
		"Name",
		"L1 Ups",
		"L1 Flaps",
		"L2 Ups",
		"L2 Flaps",
	)

	for _, d := range c.Devices().Items() {
		isisMetric := otg.Telemetry().IsisRouter(d.Isis().Name()).Get(t)
		name := isisMetric.GetName()
		l1Ups := isisMetric.GetCounters().GetLevel1().GetSessionsUp()
		l1Flaps := isisMetric.GetCounters().GetLevel1().GetSessionsFlap()
		l2Ups := isisMetric.GetCounters().GetLevel2().GetSessionsUp()
		l2Flaps := isisMetric.GetCounters().GetLevel2().GetSessionsFlap()
		out.WriteString(fmt.Sprintf(
			"%-15v%-15v%-15v%-15v%-15v\n",
			name, l1Ups, l1Flaps, l2Ups, l2Flaps,
		))
	}
	fmt.Fprintln(&out, strings.Repeat("-", 130))
	out.WriteString("\n\n")
	t.Log(out.String())
}
