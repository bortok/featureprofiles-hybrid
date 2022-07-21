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
		"Lag",
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
