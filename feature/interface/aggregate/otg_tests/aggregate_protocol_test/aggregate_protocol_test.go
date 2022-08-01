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

package rt_5_2_aggregate_lacp_protocol

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

type OTGBGPMetric struct {
	State string
}

type OTGIsIsMetric struct {
	L1Ups uint64
	L2Ups uint64
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
		for memberPort := range expectedLacpMembers {
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

func verifyOTGBGPTelemetry(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedBGPMetric map[string]OTGBGPMetric) {
	for _, d := range config.Devices().Items() {
		for _, ip := range d.Bgp().Ipv4Interfaces().Items() {
			for _, configPeer := range ip.Peers().Items() {
				nbrPath := otg.Telemetry().BgpPeer(configPeer.Name())
				_, ok := nbrPath.SessionState().Watch(t, time.Minute,
					func(val *otgtelemetry.QualifiedE_BgpPeer_SessionState) bool {
						return val.IsPresent() && val.Val(t).String() == expectedBGPMetric[configPeer.Name()].State
					}).Await(t)
				if !ok {
					otgutils.LogBGPv4Metrics(t, otg, config)
					t.Fatal(t, "for BGP Peer ", configPeer.Name(), " State is: ", nbrPath.SessionState().Get(t).String())
				}
			}
		}
	}
}

func verifyOTGIsIsTelemetry(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedIsIsMetric map[string]OTGIsIsMetric) {
	for _, d := range config.Devices().Items() {
		routerPath := otg.Telemetry().IsisRouter(d.Isis().Name())
		_, ok := routerPath.Counters().Level1().SessionsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedIsIsMetric[d.Isis().Name()].L1Ups
			}).Await(t)
		if !ok {
			otgutils.LogISISMetrics(t, otg, config)
			t.Fatal(t, "for ISIS Router ", d.Isis().Name(), " L1 Session UP is: ", routerPath.Counters().Level1().SessionsUp().Get(t))
		}
		_, ok = routerPath.Counters().Level2().SessionsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedIsIsMetric[d.Isis().Name()].L2Ups
			}).Await(t)
		if !ok {
			otgutils.LogISISMetrics(t, otg, config)
			t.Fatal(t, "for ISIS Router ", d.Isis().Name(), " L2 Session UP is: ", routerPath.Counters().Level2().SessionsUp().Get(t))
		}
	}
}

func TestAggregateLacpProtocol(t *testing.T) {
	dut := ondatra.DUT(t, "dut")
	configureDUT(t, dut)
	defer unsetDUT(t, dut)

	ate := ondatra.ATE(t, "ate")
	otg := ate.OTG()
	config := configureOTG(t, otg)

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

	expectedBGPMetric := map[string]OTGBGPMetric{
		"p1d1.bgp1":   {State: "ESTABLISHED"},
		"lag1d1.bgp1": {State: "ESTABLISHED"},
	}
	t.Logf("Checking BGP metrics as expected on OTG")
	verifyOTGBGPTelemetry(t, otg, config, expectedBGPMetric)
	otgutils.LogBGPv4Metrics(t, otg, config)

	expectedIsIsMetric := map[string]OTGIsIsMetric{
		"p1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
		"lag1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
	}
	t.Logf("Checking ISIS metrics as expected on OTG")
	verifyOTGIsIsTelemetry(t, otg, config, expectedIsIsMetric)
	otgutils.LogISISMetrics(t, otg, config)

	// as up links >  min links
	fmt.Println("Making LAG Member port2-4 down")
	otg.DownLacpMember(t, []string{"port2", "port3", "port4"})

	t.Logf("Check Interface status on DUT after bringing 3 of 8 port links down (up links > min links) ")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
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

	bgpHoldTime := config.Devices().Items()[1].Bgp().Ipv4Interfaces().Items()[0].Peers().Items()[0].Advanced().HoldTimeInterval() + 5
	t.Logf("Waiting for BGP hold interval to over...")
	time.Sleep(time.Duration(bgpHoldTime) * time.Second)

	expectedBGPMetric = map[string]OTGBGPMetric{
		"p1d1.bgp1":   {State: "ESTABLISHED"},
		"lag1d1.bgp1": {State: "ESTABLISHED"},
	}
	t.Logf("Checking BGP metrics as expected on OTG")
	verifyOTGBGPTelemetry(t, otg, config, expectedBGPMetric)
	otgutils.LogBGPv4Metrics(t, otg, config)

	expectedIsIsMetric = map[string]OTGIsIsMetric{
		"p1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
		"lag1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
	}
	t.Logf("Checking ISIS metrics as expected on OTG")
	verifyOTGIsIsTelemetry(t, otg, config, expectedIsIsMetric)
	otgutils.LogISISMetrics(t, otg, config)

	// as up links =  min links
	fmt.Println("Making LAG Member port5 down")
	otg.DownLacpMember(t, []string{"port5"})

	t.Logf("Check Interface status on DUT after 4 of 8 port links down (up links = min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port5").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
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

	t.Logf("Waiting for BGP hold interval to over...")
	time.Sleep(time.Duration(bgpHoldTime) * time.Second)
	expectedBGPMetric = map[string]OTGBGPMetric{
		"p1d1.bgp1":   {State: "ESTABLISHED"},
		"lag1d1.bgp1": {State: "ESTABLISHED"},
	}
	t.Logf("Checking BGP metrics as expected on OTG")
	verifyOTGBGPTelemetry(t, otg, config, expectedBGPMetric)
	otgutils.LogBGPv4Metrics(t, otg, config)

	expectedIsIsMetric = map[string]OTGIsIsMetric{
		"p1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
		"lag1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
	}
	t.Logf("Checking ISIS metrics as expected on OTG")
	verifyOTGIsIsTelemetry(t, otg, config, expectedIsIsMetric)
	otgutils.LogISISMetrics(t, otg, config)

	// as up links < min links
	fmt.Println("Making LAG Member port6 down ")
	otg.DownLacpMember(t, []string{"port6"})

	t.Logf("Check Interface status on DUT after 5 of 8 port links down (up links < min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "LOWER_LAYER_DOWN")

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

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port5").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port6").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port7").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port8").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port9").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
		},
	}

	t.Logf("Check LACP Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedBGPMetric = map[string]OTGBGPMetric{
		"p1d1.bgp1":   {State: "ESTABLISHED"},
		"lag1d1.bgp1": {State: "IDLE"},
	}
	t.Logf("Checking BGP metrics as expected on OTG")
	verifyOTGBGPTelemetry(t, otg, config, expectedBGPMetric)
	otgutils.LogBGPv4Metrics(t, otg, config)

	expectedIsIsMetric = map[string]OTGIsIsMetric{
		"p1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
		"lag1d1.isis1": {
			L1Ups: 0,
			L2Ups: 0,
		},
	}
	t.Logf("Checking ISIS metrics as expected on OTG")
	verifyOTGIsIsTelemetry(t, otg, config, expectedIsIsMetric)
	otgutils.LogISISMetrics(t, otg, config)

	// as up links = min links
	fmt.Println("Making LAG Member port6 up ")
	otg.UpLacpMember(t, []string{"port6"})

	t.Logf("Check Interface status on DUT after 4 of 8 port links down (up links = min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

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

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 4},
	}

	t.Logf("Checking LAG metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port5").Name(): {Synchronization: "IN_SYNC", Collecting: false, Distributing: false},
			dut.Port(t, "port6").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port7").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port8").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
			dut.Port(t, "port9").Name(): {Synchronization: "IN_SYNC", Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check LACP Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	t.Logf("Waiting for BGP hold interval to over...")
	time.Sleep(time.Duration(bgpHoldTime) * time.Second)

	expectedBGPMetric = map[string]OTGBGPMetric{
		"p1d1.bgp1":   {State: "ESTABLISHED"},
		"lag1d1.bgp1": {State: "ESTABLISHED"},
	}
	t.Logf("Checking BGP metrics as expected on OTG")
	verifyOTGBGPTelemetry(t, otg, config, expectedBGPMetric)
	otgutils.LogBGPv4Metrics(t, otg, config)

	expectedIsIsMetric = map[string]OTGIsIsMetric{
		"p1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
		"lag1d1.isis1": {
			L1Ups: 1,
			L2Ups: 0,
		},
	}
	t.Logf("Checking ISIS metrics as expected on OTG")
	verifyOTGIsIsTelemetry(t, otg, config, expectedIsIsMetric)
	otgutils.LogISISMetrics(t, otg, config)

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

	p1d1eth1ip1 := p1d1eth1.Ipv4Addresses().Add().
		SetName("p1d1.eth1.ip1").
		SetAddress("11.1.1.2").
		SetGateway("11.1.1.1").
		SetPrefix(24)

	// BGP on device 1
	p1d1inf := p1d1.Bgp().SetRouterId("11.1.1.2").Ipv4Interfaces().Add().
		SetIpv4Name(p1d1eth1ip1.Name())

	p1d1infpeer1 := p1d1inf.Peers().Add().
		SetName("p1d1.bgp1").
		SetPeerAddress("11.1.1.1").
		SetAsNumber(65100).
		SetAsType("ebgp")

	p1d1infpeer1.Advanced().SetKeepAliveInterval(5).SetHoldTimeInterval(15)

	p1d1infpeer1v4 := p1d1infpeer1.V4Routes().Add().
		SetName("p1d1.bgp1.v4")
	p1d1infpeer1v4.Advanced().
		SetMultiExitDiscriminator(50).
		SetOrigin("egp")

	p1d1infpeer1v4.Addresses().Add().
		SetAddress("111.1.0.1").
		SetPrefix(32).
		SetCount(1).
		SetStep(1)

	p1d1infpeer1v4.Communities().Add().
		SetAsCustom(2).
		SetAsNumber(1).
		SetType("manual_as_number")

	p1d1infpeer1v4.AsPath().
		SetAsSetMode("include_as_set").
		Segments().Add().
		SetAsNumbers([]int64{1, 2}).
		SetType("as_seq")

	// ISIS on device 1
	p1d1isis1 := p1d1.Isis().
		SetName("p1d1.isis1").
		SetSystemId("640000000001")
	p1d1isis1.Basic().SetIpv4TeRouterId(p1d1eth1ip1.Address())
	p1d1isis1.Basic().SetHostname("ixia-c-port1")
	p1d1isis1.Basic().SetEnableWideMetric(true)
	p1d1isis1.Advanced().SetAreaAddresses([]string{"490002"})
	p1d1isis1.Advanced().SetCsnpInterval(10000)
	p1d1isis1.Advanced().SetEnableHelloPadding(true)
	p1d1isis1.Advanced().SetLspLifetime(1200)
	p1d1isis1.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d1isis1.Advanced().SetLspRefreshRate(900)
	p1d1isis1.Advanced().SetMaxAreaAddresses(3)
	p1d1isis1.Advanced().SetMaxLspSize(1492)
	p1d1isis1.Advanced().SetPsnpInterval(2000)
	p1d1isis1.Advanced().SetEnableAttachedBit(false)
	//ISIS interface
	p1d1isis1intf1 := p1d1isis1.Interfaces().
		Add().
		SetName("p1d1.isis1.intf1").
		SetEthName(p1d1eth1.Name()).
		SetNetworkType(gosnappi.IsisInterfaceNetworkType.POINT_TO_POINT).
		SetLevelType(gosnappi.IsisInterfaceLevelType.LEVEL_1).
		SetMetric(10)
	//ISIS L1 settings
	p1d1isis1intf1.
		L1Settings().
		SetDeadInterval(15).
		SetHelloInterval(5).
		SetPriority(0)
	//ISIS advanced settings
	p1d1isis1intf1.
		Advanced().SetAutoAdjustSupportedProtocols(true).SetAutoAdjustArea(true).SetAutoAdjustMtu(true)

	// device on lag1
	lag1d1 := config.Devices().Add().SetName("lag1d1")

	lag1d1eth1 := lag1d1.Ethernets().Add().
		SetName("lag1d1.eth1").
		SetPortName(lag1.Name()).
		SetMac("00:22:01:00:00:01").
		SetMtu(1500)

	lag1d1eth1ip1 := lag1d1eth1.Ipv4Addresses().Add().
		SetName("lag1d1.eth1.ip1").
		SetAddress("21.1.1.2").
		SetGateway("21.1.1.1").
		SetPrefix(24)

	// BGP on device2
	lag1d1inf := lag1d1.Bgp().SetRouterId("21.1.1.2").Ipv4Interfaces().Add().
		SetIpv4Name(lag1d1eth1ip1.Name())

	lag1d1infpeer1 := lag1d1inf.Peers().Add().
		SetName("lag1d1.bgp1").
		SetPeerAddress("21.1.1.1").
		SetAsNumber(65300).
		SetAsType("ebgp")

	lag1d1infpeer1.Advanced().SetKeepAliveInterval(5).SetHoldTimeInterval(15)

	lag1d1infpeer1v4 := lag1d1infpeer1.V4Routes().Add().
		SetName("lag1d1.bgp1.v4")
	lag1d1infpeer1v4.Advanced().
		SetMultiExitDiscriminator(50).
		SetOrigin("egp")

	lag1d1infpeer1v4.Addresses().Add().
		SetAddress("211.1.0.1").
		SetPrefix(32).
		SetCount(10).
		SetStep(1)

	lag1d1infpeer1v4.Communities().Add().
		SetAsCustom(2).
		SetAsNumber(1).
		SetType("manual_as_number")

	lag1d1infpeer1v4.AsPath().
		SetAsSetMode("include_as_set").
		Segments().Add().
		SetAsNumbers([]int64{100, 200}).
		SetType("as_seq")

	// ISIS on device 2
	lag1d1isis1 := lag1d1.Isis().
		SetName("lag1d1.isis1").
		SetSystemId("650000000001")
	lag1d1isis1.Basic().SetIpv4TeRouterId(lag1d1eth1ip1.Address())
	lag1d1isis1.Basic().SetHostname("ixia-c-port2")
	lag1d1isis1.Basic().SetEnableWideMetric(true)
	lag1d1isis1.Advanced().SetAreaAddresses([]string{"490002"})
	lag1d1isis1.Advanced().SetCsnpInterval(10000)
	lag1d1isis1.Advanced().SetEnableHelloPadding(true)
	lag1d1isis1.Advanced().SetLspLifetime(1200)
	lag1d1isis1.Advanced().SetLspMgroupMinTransInterval(5000)
	lag1d1isis1.Advanced().SetLspRefreshRate(900)
	lag1d1isis1.Advanced().SetMaxAreaAddresses(3)
	lag1d1isis1.Advanced().SetMaxLspSize(1492)
	lag1d1isis1.Advanced().SetPsnpInterval(2000)
	lag1d1isis1.Advanced().SetEnableAttachedBit(false)
	//isis interface
	lag1d1isis1intf1 := lag1d1isis1.Interfaces().
		Add().
		SetName("lag1d1.isis1.intf1").
		SetEthName(lag1d1eth1.Name()).
		SetNetworkType(gosnappi.IsisInterfaceNetworkType.POINT_TO_POINT).
		SetLevelType(gosnappi.IsisInterfaceLevelType.LEVEL_1).
		SetMetric(10)
	//isis L1 settings
	lag1d1isis1intf1.
		L1Settings().
		SetDeadInterval(15).
		SetHelloInterval(5).
		SetPriority(0)
	//isis advanced settings
	lag1d1isis1intf1.
		Advanced().SetAutoAdjustSupportedProtocols(true).SetAutoAdjustArea(true).SetAutoAdjustMtu(true)

	return config
}
