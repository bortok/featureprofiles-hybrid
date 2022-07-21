// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rt_5_2_aggregate_lacp

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/featureprofiles/internal/fptest"
	"github.com/openconfig/featureprofiles/internal/otgutils"
	"github.com/openconfig/ondatra"
	"github.com/openconfig/ondatra/otg"
	"github.com/openconfig/ondatra/telemetry"
	otgtelemetry "github.com/openconfig/ondatra/telemetry/otg"
)

type DUTLacpMember struct {
	Collecting      bool
	Distributing    bool
	Synchronization string
}

type OtgLagMetric struct {
	Status        string
	MemberPortsUp int32
}

type OtgLacpMetric struct {
	Collecting      bool
	Distributing    bool
	Synchronization string
}

func TestMain(m *testing.M) {
	fptest.RunTests(m)
}

func configureDUT(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Configuring DUT...")
	dut.Config().New().WithAristaFile("set_arista.config").Push(t)
}

func unsetDUT(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Configuring DUT...")
	dut.Config().New().WithAristaFile("unset_arista.config").Push(t)
}

func getInterfaceMacs(t *testing.T, dut *ondatra.DUTDevice) map[string]string {
	dutMacDetails := make(map[string]string)
	for _, p := range dut.Ports() {
		eth := dut.Telemetry().Interface(p.Name()).Ethernet().Get(t)
		t.Logf("Mac address of Interface %s in DUT: %s", p.Name(), eth.GetMacAddress())
		dutMacDetails[p.Name()] = eth.GetMacAddress()
	}
	return dutMacDetails
}

func dutVerifyInterfaceStatus(t *testing.T, dut *ondatra.DUTDevice, interfaceName string, expStatus string) {
	interfacePath := dut.Telemetry().Interface(interfaceName)
	_, ok := interfacePath.OperStatus().Watch(t, time.Minute,
		func(val *telemetry.QualifiedE_Interface_OperStatus) bool {
			return val.IsPresent() && val.Val(t).String() == expStatus
		}).Await(t)
	if !ok {
		t.Fatal(t, "Interface reported Oper status", interfacePath.OperStatus().Get(t).String())
	} else {
		t.Logf("Interface %s is %s", interfaceName, expStatus)
	}
}

func logDUTLacpMetrics(t testing.TB, dut *ondatra.DUTDevice, expectedDUTLacpMember map[string]map[string]DUTLacpMember) {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nDUT LACP Metrics\n")
	for i := 1; i <= 120; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out,
		"%-20s%-20s%-20s%-20s%-20s\n",
		"Port Channel",
		"Member Interface",
		"Synchronization",
		"Collecting",
		"Distributing",
	)
	for iFace, expectedLacpMembers := range expectedDUTLacpMember {
		lacpInterfacePath := dut.Telemetry().Lacp().Interface(iFace)
		for memberPort, _ := range expectedLacpMembers {
			memberPortStat := lacpInterfacePath.Member(memberPort).Get(t)
			out.WriteString(fmt.Sprintf(
				"%-20v%-20v%-20v%-20v%-20v\n",
				iFace, memberPort, memberPortStat.GetSynchronization().String(), memberPortStat.GetCollecting(), memberPortStat.GetDistributing(),
			))

		}
	}
	fmt.Fprintln(&out, strings.Repeat("-", 120))
	out.WriteString("\n\n")
	t.Log(out.String())
}

func dutLacpMemberPortsAsExpected(t *testing.T, dut *ondatra.DUTDevice, expectedDUTLacpMember map[string]map[string]DUTLacpMember) (bool, error) {
	for iFace, expectedLacpMembers := range expectedDUTLacpMember {
		lacpInterfacePath := dut.Telemetry().Lacp().Interface(iFace)
		for memberPort, expectedInfo := range expectedLacpMembers {
			memberPath := lacpInterfacePath.Member(memberPort)
			_, ok := memberPath.Collecting().Watch(t, time.Minute,
				func(val *telemetry.QualifiedBool) bool {
					return val.IsPresent() && val.Val(t) == expectedInfo.Collecting
				}).Await(t)
			if !ok {
				logDUTLacpMetrics(t, dut, expectedDUTLacpMember)
				t.Fatal(t, "LACP Member Port ", memberPort, " Collecting is ", memberPath.Collecting().Get(t))
			}

			_, ok = memberPath.Distributing().Watch(t, time.Minute,
				func(val *telemetry.QualifiedBool) bool {
					return val.IsPresent() && val.Val(t) == expectedInfo.Distributing
				}).Await(t)
			if !ok {
				logDUTLacpMetrics(t, dut, expectedDUTLacpMember)
				t.Fatal(t, "LACP Member Port ", memberPort, " Distributing is ", memberPath.Distributing().Get(t))
			}

			_, ok = memberPath.Synchronization().Watch(t, time.Minute,
				func(val *telemetry.QualifiedE_Lacp_LacpSynchronizationType) bool {
					return val.IsPresent() && val.Val(t).String() == expectedInfo.Synchronization
				}).Await(t)
			if !ok {
				logDUTLacpMetrics(t, dut, expectedDUTLacpMember)
				t.Fatal(t, "LACP Member Port ", memberPort, " Synchronization is ", memberPath.Synchronization().Get(t).String())
			}
		}
	}
	logDUTLacpMetrics(t, dut, expectedDUTLacpMember)
	return true, nil
}

func otgLagAsExpected(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedOtgLagMetrics map[string]OtgLagMetric) {
	for lag, expOtgLagMetric := range expectedOtgLagMetrics {
		lagPath := otg.Telemetry().Lag(lag)
		_, ok := lagPath.OperStatus().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedE_Lag_OperStatus) bool {
				return val.IsPresent() && val.Val(t).String() == expOtgLagMetric.Status
			}).Await(t)
		if !ok {
			otgutils.LogLagMetrics(t, otg, config)
			t.Fatal(t, "for LAG ", lag, " Oper Status: ", lagPath.OperStatus().Get(t))
		}

		_, ok = lagPath.Counters().MemberPortsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == uint64(expOtgLagMetric.MemberPortsUp)
			}).Await(t)
		if !ok {
			otgutils.LogLagMetrics(t, otg, config)
			t.Fatal(t, "For LAG ", lag, " Member Ports Up Count: ", lagPath.Counters().MemberPortsUp().Get(t))
		}
	}
}

func otgLacpAsExpected(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedOtgLacpMetrics map[string]OtgLacpMetric) {
	for lacpMemberPort, expOtgLacpMetric := range expectedOtgLacpMetrics {
		lacpMemberPath := otg.Telemetry().Lacp().LagMember(lacpMemberPort)
		_, ok := lacpMemberPath.Collecting().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedBool) bool {
				return val.IsPresent() && val.Val(t) == expOtgLacpMetric.Collecting
			}).Await(t)
		if !ok {
			otgutils.LogLacpMetrics(t, otg, config)
			t.Fatal(t, "for LACP Port ", lacpMemberPort, " Collecting is: ", lacpMemberPath.Collecting().Get(t))
		}

		_, ok = lacpMemberPath.Distributing().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedBool) bool {
				return val.IsPresent() && val.Val(t) == expOtgLacpMetric.Distributing
			}).Await(t)
		if !ok {
			otgutils.LogLacpMetrics(t, otg, config)
			t.Fatal(t, "for LACP Port ", lacpMemberPort, " Distributing is: ", lacpMemberPath.Distributing().Get(t))
		}

		_, ok = lacpMemberPath.Synchronization().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedE_OpenTrafficGeneratorLacp_LacpSynchronizationType) bool {
				return val.IsPresent() && val.Val(t).String() == expOtgLacpMetric.Synchronization
			}).Await(t)
		if !ok {
			otgutils.LogLacpMetrics(t, otg, config)
			t.Fatal(t, "for LACP Port ", lacpMemberPort, " Synchronization is: ", lacpMemberPath.Synchronization().Get(t).String())
		}
	}
}

func waitFor(fn func() bool, t testing.TB, interval time.Duration, timeout time.Duration) {
	start := time.Now()
	for {
		done := fn()
		if done {
			t.Logf("Expected stats received...")
			break
		}
		if time.Since(start) > timeout {
			t.Fatal("Timeout while waiting for expected stats...")
			break
		}
		time.Sleep(interval)
	}
}

func aggregateFlowMetricsAsExpected(t testing.TB, otg *otg.OTG, c gosnappi.Config, expectedRxPkts int) bool {
	t.Helper()
	var out strings.Builder
	out.WriteString("\nFlow Metrics\n")
	for i := 1; i <= 80; i++ {
		out.WriteString("-")
	}
	out.WriteString("\n")
	fmt.Fprintf(&out, "%-25v%-15v%-15v\n", "Name", "Frames Tx", "Frames Rx")
	totalRxPkts := 0
	totalTxPkts := 0
	expectedTxPkts := 0
	for _, f := range c.Flows().Items() {
		expectedTxPkts += int(f.Duration().FixedPackets().Packets())
		flowMetrics := otg.Telemetry().Flow(f.Name()).Get(t)
		totalRxPkts = totalRxPkts + int(flowMetrics.GetCounters().GetInPkts())
		totalTxPkts = totalTxPkts + int(flowMetrics.GetCounters().GetOutPkts())
	}
	out.WriteString(fmt.Sprintf("%-25v%-15v%-15v\n", "Aggregated Flow", totalTxPkts, totalRxPkts))
	fmt.Fprintln(&out, strings.Repeat("-", 80))
	out.WriteString("\n\n")
	t.Log(out.String())
	return totalRxPkts == expectedRxPkts && totalTxPkts == expectedTxPkts
}

func TestAggregateLacpTraffic(t *testing.T) {
	dut := ondatra.DUT(t, "dut")
	configureDUT(t, dut)
	defer unsetDUT(t, dut)

	ate := ondatra.ATE(t, "ate")
	otg := ate.OTG()
	config := configureOTG(t, otg)

	dutMacDetails := getInterfaceMacs(t, dut)
	config.Flows().Items()[0].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[1].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[2].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[3].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[4].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[5].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[6].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[7].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])

	otg.PushConfig(t, config)
	otg.StartProtocols(t)
	defer otg.StopProtocols(t)

	// Right After SetConfig
	t.Logf("Check Interface status on DUT")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedLacpMemberPortsMap := map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port3").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port4").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port5").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port6").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port7").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port8").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port9").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check LACP Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLacpMetrics := map[string]OtgLacpMetric{
		"port2": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port3": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port4": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port5": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port6": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port7": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port8": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port9": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
	}

	t.Logf("Checking LACP metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	expectedOtgLagMetrics := map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 8},
	}

	t.Logf("Checking LAG metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	t.Logf("Starting Traffic...")
	otg.StartTraffic(t)

	t.Logf("Waiting for flow metrics to be as expected...")
	waitFor(
		func() bool { return aggregateFlowMetricsAsExpected(t, otg, config, 400) },
		t,
		500*time.Millisecond,
		3*time.Second,
	)

	t.Logf("Stopping Traffic...")
	otg.StopTraffic(t)

	// as up links >  min links
	fmt.Println("Making LAG Member port2-4 down")
	otg.DownLacpMember(t, []string{"port2", "port3", "port4"})

	t.Logf("Check Interface status on DUT after bringing 3 of 8 port links down (up links > min links) ")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port5").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port6").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port7").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port8").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port9").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check LACP Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 5},
	}

	t.Logf("Checking LAG metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
		"port2": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port3": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port4": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port5": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port6": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port7": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port8": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port9": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
	}

	t.Logf("Checking LACP metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	t.Logf("Starting Traffic...")
	otg.StartTraffic(t)

	t.Logf("Waiting for flow metrics to be as expected...")
	waitFor(
		func() bool { return aggregateFlowMetricsAsExpected(t, otg, config, 400) },
		t,
		500*time.Millisecond,
		3*time.Second,
	)

	t.Logf("Stopping Traffic...")
	otg.StopTraffic(t)

	// as up links =  min links
	fmt.Println("Making LAG Member port5 down")
	otg.DownLacpMember(t, []string{"port5"})

	t.Logf("Check Interface status on DUT after 4 of 8 port links down (up links = min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port5").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port6").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port7").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port8").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port9").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check LACP Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 4},
	}

	t.Logf("Checking LAG metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
		"port2": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port3": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port4": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port5": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port6": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port7": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port8": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
		"port9": {
			Synchronization: "IN_SYNC",
			Collecting:      true,
			Distributing:    true,
		},
	}

	t.Logf("Checking LACP metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	t.Logf("Starting Traffic...")
	otg.StartTraffic(t)

	t.Logf("Waiting for flow metrics to be as expected...")
	waitFor(
		func() bool { return aggregateFlowMetricsAsExpected(t, otg, config, 400) },
		t,
		500*time.Millisecond,
		3*time.Second,
	)

	t.Logf("Stopping Traffic...")
	otg.StopTraffic(t)

	// as up links < min links
	fmt.Println("Making LAG Member port6 down ")
	otg.DownLacpMember(t, []string{"port6"})

	t.Logf("Check Interface status on DUT after 5 of 8 port links down (up links < min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "LOWER_LAYER_DOWN")

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port5").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port6").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port7").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port8").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port9").Name(): {Synchronization: "OUT_SYNC", Collecting: false, Distributing: false},
		},
	}

	t.Logf("Check LACP Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
		"port2": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port3": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port4": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port5": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port6": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port7": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port8": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
		"port9": {
			Synchronization: "OUT_SYNC",
			Collecting:      false,
			Distributing:    false,
		},
	}

	t.Logf("Checking LACP metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "DOWN", MemberPortsUp: 0},
	}

	t.Logf("Checking LAG metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	t.Logf("Starting Traffic...")
	otg.StartTraffic(t)

	t.Logf("Waiting for flow metrics to be as expected...")
	waitFor(
		func() bool { return aggregateFlowMetricsAsExpected(t, otg, config, 0) },
		t,
		500*time.Millisecond,
		3*time.Second,
	)

	t.Logf("Stopping Traffic...")
	otg.StopTraffic(t)
}

func configureOTG(t *testing.T, otg *otg.OTG) gosnappi.Config {
	config := otg.NewConfig(t)
	port1 := config.Ports().Add().SetName("port1")
	port2 := config.Ports().Add().SetName("port2")
	port3 := config.Ports().Add().SetName("port3")
	port4 := config.Ports().Add().SetName("port4")
	port5 := config.Ports().Add().SetName("port5")
	port6 := config.Ports().Add().SetName("port6")
	port7 := config.Ports().Add().SetName("port7")
	port8 := config.Ports().Add().SetName("port8")
	port9 := config.Ports().Add().SetName("port9")

	// lag1
	lag1 := config.Lags().Add().SetName("lag1")

	lag1.SetMinLinks(4)
	lag1.Protocol().Lacp().SetActorKey(1).SetActorSystemId("01:01:01:01:01:01").SetActorSystemPriority(1)

	// port2 as port of lag1
	lag1port1 := lag1.Ports().Add().
		SetPortName(port2.Name())
	lag1port1.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(1).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port1.Ethernet().SetMac("00:00:00:00:00:16").SetName("lag1.port1.eth")

	// port3 as port of lag1
	lag1port2 := lag1.Ports().Add().
		SetPortName(port3.Name())
	lag1port2.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(2).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port2.Ethernet().SetMac("00:00:00:00:00:17").SetName("lag1.port2.eth")

	// port4 as port of lag1
	lag1port3 := lag1.Ports().Add().
		SetPortName(port4.Name())
	lag1port3.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(3).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port3.Ethernet().SetMac("00:00:00:00:00:18").SetName("lag1.port3.eth")

	// port5 as port of lag1
	lag1port4 := lag1.Ports().Add().
		SetPortName(port5.Name())
	lag1port4.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(4).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port4.Ethernet().SetMac("00:00:00:00:00:19").SetName("lag1.port4.eth")

	// port6 as port of lag1
	lag1port5 := lag1.Ports().Add().
		SetPortName(port6.Name())
	lag1port5.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(5).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port5.Ethernet().SetMac("00:00:00:00:00:20").SetName("lag1.port5.eth")

	// port7 as port of lag1
	lag1port6 := lag1.Ports().Add().
		SetPortName(port7.Name())
	lag1port6.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(6).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port6.Ethernet().SetMac("00:00:00:00:00:21").SetName("lag1.port6.eth")

	// port8 as port of lag1
	lag1port7 := lag1.Ports().Add().
		SetPortName(port8.Name())
	lag1port7.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(7).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port7.Ethernet().SetMac("00:00:00:00:00:22").SetName("lag1.port7.eth")

	// port9 as port of lag1
	lag1port8 := lag1.Ports().Add().
		SetPortName(port9.Name())
	lag1port8.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(8).
		SetActorPortPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(3)
	lag1port8.Ethernet().SetMac("00:00:00:00:00:23").SetName("lag1.port8.eth")

	// device on port1
	p1d1 := config.Devices().Add().SetName("p1d1")

	p1d1eth1 := p1d1.Ethernets().Add().
		SetName("p1d1.eth1").
		SetPortName(port1.Name()).
		SetMac("00:11:01:00:00:01").
		SetMtu(1500)

	p1d1eth1.Ipv4Addresses().Add().
		SetName("p1d1.eth1.ip1").
		SetAddress("11.1.1.2").
		SetGateway("11.1.1.1").
		SetPrefix(24)

	// device on lag1
	lag1d1 := config.Devices().Add().SetName("lag1d1")

	lag1d1eth1 := lag1d1.Ethernets().Add().
		SetName("lag1d1.eth1").
		SetPortName(lag1.Name()).
		SetMac("00:22:01:00:00:01").
		SetMtu(1500)

	lag1d1eth1.Ipv4Addresses().Add().
		SetName("lag1d1.eth1.ip1").
		SetAddress("21.1.1.2").
		SetGateway("21.1.1.1").
		SetPrefix(24)

	// flow port1 -> port2
	flow1 := config.Flows().Add().SetName("port1->port2")
	flow1.Metrics().SetEnable(true)
	flow1.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port2.Name())
	flow1.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow1.Size().SetChoice("fixed").SetFixed(128)
	flow1.Rate().SetChoice("pps").SetPps(50)
	flow1Eth := flow1.Packet().Add().SetChoice("ethernet").Ethernet()
	flow1Eth.Dst().SetChoice("value")
	flow1Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow1IP := flow1.Packet().Add().Ipv4()
	flow1IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow1IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port3
	flow2 := config.Flows().Add().SetName("port1->port3")
	flow2.Metrics().SetEnable(true)
	flow2.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port3.Name())
	flow2.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow2.Size().SetChoice("fixed").SetFixed(128)
	flow2.Rate().SetChoice("pps").SetPps(50)
	flow2Eth := flow2.Packet().Add().SetChoice("ethernet").Ethernet()
	flow2Eth.Dst().SetChoice("value")
	flow2Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow2IP := flow2.Packet().Add().Ipv4()
	flow2IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow2IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port4
	flow3 := config.Flows().Add().SetName("port1->port4")
	flow3.Metrics().SetEnable(true)
	flow3.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port4.Name())
	flow3.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow3.Size().SetChoice("fixed").SetFixed(128)
	flow3.Rate().SetChoice("pps").SetPps(50)
	flow3Eth := flow3.Packet().Add().SetChoice("ethernet").Ethernet()
	flow3Eth.Dst().SetChoice("value")
	flow3Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow3IP := flow3.Packet().Add().Ipv4()
	flow3IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow3IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port5
	flow4 := config.Flows().Add().SetName("port1->port5")
	flow4.Metrics().SetEnable(true)
	flow4.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port5.Name())
	flow4.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow4.Size().SetChoice("fixed").SetFixed(128)
	flow4.Rate().SetChoice("pps").SetPps(50)
	flow4Eth := flow4.Packet().Add().SetChoice("ethernet").Ethernet()
	flow4Eth.Dst().SetChoice("value")
	flow4Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow4IP := flow4.Packet().Add().Ipv4()
	flow4IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow4IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port6
	flow5 := config.Flows().Add().SetName("port1->port6")
	flow5.Metrics().SetEnable(true)
	flow5.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port6.Name())
	flow5.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow5.Size().SetChoice("fixed").SetFixed(128)
	flow5.Rate().SetChoice("pps").SetPps(50)
	flow5Eth := flow5.Packet().Add().SetChoice("ethernet").Ethernet()
	flow5Eth.Dst().SetChoice("value")
	flow5Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow5IP := flow5.Packet().Add().Ipv4()
	flow5IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow5IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port7
	flow6 := config.Flows().Add().SetName("port1->port7")
	flow6.Metrics().SetEnable(true)
	flow6.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port7.Name())
	flow6.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow6.Size().SetChoice("fixed").SetFixed(128)
	flow6.Rate().SetChoice("pps").SetPps(50)
	flow6Eth := flow6.Packet().Add().SetChoice("ethernet").Ethernet()
	flow6Eth.Dst().SetChoice("value")
	flow6Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow6IP := flow6.Packet().Add().Ipv4()
	flow6IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow6IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port8
	flow7 := config.Flows().Add().SetName("port1->port8")
	flow7.Metrics().SetEnable(true)
	flow7.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port8.Name())
	flow7.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow7.Size().SetChoice("fixed").SetFixed(128)
	flow7.Rate().SetChoice("pps").SetPps(50)
	flow7Eth := flow7.Packet().Add().SetChoice("ethernet").Ethernet()
	flow7Eth.Dst().SetChoice("value")
	flow7Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow7IP := flow7.Packet().Add().Ipv4()
	flow7IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow7IP.Src().SetChoice("value").SetValue("11.1.1.2")

	// flow port1 -> port9
	flow8 := config.Flows().Add().SetName("port1->port9")
	flow8.Metrics().SetEnable(true)
	flow8.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port9.Name())
	flow8.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(50)
	flow8.Size().SetChoice("fixed").SetFixed(128)
	flow8.Rate().SetChoice("pps").SetPps(50)
	flow8Eth := flow8.Packet().Add().SetChoice("ethernet").Ethernet()
	flow8Eth.Dst().SetChoice("value")
	flow8Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow8IP := flow8.Packet().Add().Ipv4()
	flow8IP.Dst().SetChoice("value").SetValue("21.1.1.2")
	flow8IP.Src().SetChoice("value").SetValue("11.1.1.2")

	return config
}
