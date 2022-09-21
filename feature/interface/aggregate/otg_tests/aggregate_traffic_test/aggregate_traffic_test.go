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

package aggregate_traffic

import (
	"fmt"
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

type OtgPortMetric struct {
	TxPackets uint64
	RxPackets uint64
}

type OtgFlowMetric struct {
	TxPackets uint64
	RxPackets uint64
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

func dutLacpMemberPortsAsExpected(t *testing.T, dut *ondatra.DUTDevice, ExpectedDUTLacpMember map[string]map[string]DUTLacpMember) (bool, error) {
	for iFace, expectedLacpMembers := range ExpectedDUTLacpMember {
		lacpInterfacePath := dut.Telemetry().Lacp().Interface(iFace)
		for memberPort, expectedInfo := range expectedLacpMembers {
			memberPath := lacpInterfacePath.Member(memberPort)
			_, ok := memberPath.Collecting().Watch(t, time.Minute,
				func(val *telemetry.QualifiedBool) bool {
					return val.IsPresent() && val.Val(t) == expectedInfo.Collecting
				}).Await(t)
			if !ok {
				t.Fatal(t, "Lacp Member Port ", memberPort, " Collecting is ", memberPath.Collecting().Get(t))
			} else {
				t.Logf("Collecting of Lacp Member Port %s is %v", memberPort, expectedInfo.Collecting)
			}

			_, ok = memberPath.Distributing().Watch(t, time.Minute,
				func(val *telemetry.QualifiedBool) bool {
					return val.IsPresent() && val.Val(t) == expectedInfo.Distributing
				}).Await(t)
			if !ok {
				t.Fatal(t, "Lacp Member Port ", memberPort, " Distributing is ", memberPath.Distributing().Get(t))
			} else {
				t.Logf("Distributing of Lacp Member Port %s is %v", memberPort, expectedInfo.Distributing)
			}

			_, ok = memberPath.Synchronization().Watch(t, time.Minute,
				func(val *telemetry.QualifiedE_Lacp_LacpSynchronizationType) bool {
					return val.IsPresent() && val.Val(t).String() == expectedInfo.Synchronization
				}).Await(t)
			if !ok {
				t.Fatal(t, "Lacp Member Port ", memberPort, " Synchronization is ", memberPath.Synchronization().Get(t).String())
			} else {
				t.Logf("Synchronization of Lacp Member Port %s is %v", memberPort, expectedInfo.Synchronization)
			}
		}
	}
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
			t.Fatal(t, "for Lag ", lag, " Oper Status: ", lagPath.OperStatus().Get(t))
		}

		_, ok = lagPath.Counters().MemberPortsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == uint64(expOtgLagMetric.MemberPortsUp)
			}).Await(t)
		if !ok {
			otgutils.LogLagMetrics(t, otg, config)
			t.Fatal(t, "For Lag ", lag, " Member Ports Up Count: ", lagPath.Counters().MemberPortsUp().Get(t))
		}
	}
}

func otgPortMetricAsExpected(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedOtgPortMetrics map[string]OtgPortMetric) {
	for port, expOtgPortMetric := range expectedOtgPortMetrics {
		portPath := otg.Telemetry().Port(port)
		_, ok := portPath.Counters().OutFrames().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expOtgPortMetric.TxPackets
			}).Await(t)
		if !ok {
			otgutils.LogPortMetrics(t, otg, config)
			t.Fatal(t, "for port ", port, " Tx Packets: ", portPath.Counters().OutFrames().Get(t), "but expected: ", expOtgPortMetric.TxPackets)
		}

		_, ok = portPath.Counters().InFrames().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) >= expOtgPortMetric.RxPackets
			}).Await(t)
		if !ok {
			otgutils.LogPortMetrics(t, otg, config)
			t.Fatal(t, "for port ", port, " Rx Packets: ", portPath.Counters().InFrames().Get(t), "but expected: ", expOtgPortMetric.RxPackets)
		}
	}
}

func otgFlowMetricAsExpected(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedOtgFlowMetrics map[string]OtgFlowMetric) {
	for flow, expOtgFlowMetric := range expectedOtgFlowMetrics {
		flowPath := otg.Telemetry().Flow(flow)
		_, ok := flowPath.Counters().OutPkts().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expOtgFlowMetric.TxPackets
			}).Await(t)
		if !ok {
			otgutils.LogPortMetrics(t, otg, config)
			t.Fatal(t, "for flow ", flow, " Tx Packets: ", flowPath.Counters().OutPkts().Get(t), "but expected: ", expOtgFlowMetric.TxPackets)
		}

		_, ok = flowPath.Counters().InPkts().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expOtgFlowMetric.RxPackets
			}).Await(t)
		if !ok {
			otgutils.LogPortMetrics(t, otg, config)
			t.Fatal(t, "for flow ", flow, " Rx Packets: ", flowPath.Counters().InPkts().Get(t), "but expected: ", expOtgFlowMetric.RxPackets)
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
			t.Fatal(t, "for Lacp Port ", lacpMemberPort, " Collecting is: ", lacpMemberPath.Collecting().Get(t))
		}

		_, ok = lacpMemberPath.Distributing().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedBool) bool {
				return val.IsPresent() && val.Val(t) == expOtgLacpMetric.Distributing
			}).Await(t)
		if !ok {
			otgutils.LogLacpMetrics(t, otg, config)
			t.Fatal(t, "for Lacp Port ", lacpMemberPort, " Distributing is: ", lacpMemberPath.Distributing().Get(t))
		}

		_, ok = lacpMemberPath.Synchronization().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedE_OpenTrafficGeneratorLacp_LacpSynchronizationType) bool {
				return val.IsPresent() && val.Val(t).String() == expOtgLacpMetric.Synchronization
			}).Await(t)
		if !ok {
			otgutils.LogLacpMetrics(t, otg, config)
			t.Fatal(t, "for Lacp Port ", lacpMemberPort, " Synchronization is: ", lacpMemberPath.Synchronization().Get(t).String())
		}
	}
}

func TestAggregateTraffic(t *testing.T) {
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

	// as up links >  min links
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

	t.Logf("Check Lacp Member status on DUT")
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

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	t.Logf("Check Interface status on DUT after 0 of 8 port links down (up links > min links)")
	expectedOtgLagMetrics := map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 8},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	otg.StartTraffic(t)
	expectedOtgPortMetrics := map[string]OtgPortMetric{
		"port1": {
			TxPackets: 0,
			RxPackets: 80,
		},
		"port2": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port3": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port4": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port5": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port6": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port7": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port8": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port9": {
			TxPackets: 10,
			RxPackets: 0,
		},
	}
	expectedOtgFlowMetrics := map[string]OtgFlowMetric{
		"lag-f1": {
			TxPackets: 80,
			RxPackets: 80,
		},
	}

	t.Logf("Check port & flow stats on OTG after 0 of 8 port links down (up links > min links)")
	otgFlowMetricAsExpected(t, otg, config, expectedOtgFlowMetrics)
	otgPortMetricAsExpected(t, otg, config, expectedOtgPortMetrics)
	otgutils.LogPortMetrics(t, otg, config)
	otgutils.LogFlowMetrics(t, otg, config)

	otg.StopTraffic(t)

	// as up links =  min links
	fmt.Println("Making Lag Member port2-5 down")
	otg.DisableLACPMembers(
		t, []string{
			"port2",
			"port3",
			"port4",
			"port5",
		})

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

	t.Logf("Check Lacp Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 4},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
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

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	t.Logf("Check Interface status on DUT after 4 of 8 port links down (up links = min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	otg.StartTraffic(t)
	expectedOtgPortMetrics = map[string]OtgPortMetric{
		"port1": {
			TxPackets: 0,
			RxPackets: 80,
		},
		"port2": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port3": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port4": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port5": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port6": {
			TxPackets: 20,
			RxPackets: 0,
		},
		"port7": {
			TxPackets: 20,
			RxPackets: 0,
		},
		"port8": {
			TxPackets: 20,
			RxPackets: 0,
		},
		"port9": {
			TxPackets: 20,
			RxPackets: 0,
		},
	}

	expectedOtgFlowMetrics = map[string]OtgFlowMetric{
		"lag-f1": {
			TxPackets: 80,
			RxPackets: 80,
		},
	}

	t.Logf("Check port & flow stats on OTG after 4 of 8 port links down (up links = min links)")
	otgFlowMetricAsExpected(t, otg, config, expectedOtgFlowMetrics)
	otgPortMetricAsExpected(t, otg, config, expectedOtgPortMetrics)
	otgutils.LogPortMetrics(t, otg, config)
	otgutils.LogFlowMetrics(t, otg, config)

	otg.StopTraffic(t)

	// as up links < min links
	fmt.Println("Making Lag Member port6 down ")
	otg.DisableLACPMembers(t, []string{"port6"})

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

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

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

	t.Logf("Check Lacp Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "DOWN", MemberPortsUp: 0},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	t.Logf("Check Interface status on DUT after 3 of 4 port links down (up links < min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "LOWER_LAYER_DOWN")

	otg.StartTraffic(t)
	expectedOtgPortMetrics = map[string]OtgPortMetric{
		"port1": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port2": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port3": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port4": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port5": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port6": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port7": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port8": {
			TxPackets: 0,
			RxPackets: 0,
		},
		"port9": {
			TxPackets: 0,
			RxPackets: 0,
		},
	}

	expectedOtgFlowMetrics = map[string]OtgFlowMetric{
		"lag-f1": {
			TxPackets: 80,
			RxPackets: 0,
		},
	}

	t.Logf("Check port & flow stats on OTG after lag is down (up links < min links)")
	otgFlowMetricAsExpected(t, otg, config, expectedOtgFlowMetrics)
	otgPortMetricAsExpected(t, otg, config, expectedOtgPortMetrics)
	otgutils.LogPortMetrics(t, otg, config)
	otgutils.LogFlowMetrics(t, otg, config)

	otg.StopTraffic(t)

	// as up links >  min links
	fmt.Println("Making Lag Member port2-6 up")
	otg.EnableLACPMembers(
		t, []string{
			"port2",
			"port3",
			"port4",
			"port5",
			"port6",
		})

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
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

	t.Logf("Check Lacp Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 8},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, config, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
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

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, config, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	t.Logf("Check Interface status on DUT after 0 of 8 port links down (up links > min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	otg.StartTraffic(t)
	expectedOtgPortMetrics = map[string]OtgPortMetric{
		"port1": {
			TxPackets: 0,
			RxPackets: 80,
		},
		"port2": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port3": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port4": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port5": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port6": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port7": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port8": {
			TxPackets: 10,
			RxPackets: 0,
		},
		"port9": {
			TxPackets: 10,
			RxPackets: 0,
		},
	}

	expectedOtgFlowMetrics = map[string]OtgFlowMetric{
		"lag-f1": {
			TxPackets: 80,
			RxPackets: 80,
		},
	}

	t.Logf("Check port & flow stats on OTG after 0 of 8 port links down (up links > min links)")
	otgFlowMetricAsExpected(t, otg, config, expectedOtgFlowMetrics)
	otgPortMetricAsExpected(t, otg, config, expectedOtgPortMetrics)
	otgutils.LogPortMetrics(t, otg, config)
	otgutils.LogFlowMetrics(t, otg, config)

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

	// flow lag1 -> port1
	flow1 := config.Flows().Add().SetName("lag-f1")
	flow1.Metrics().SetEnable(true)
	flow1.TxRx().SetChoice("port").Port().SetTxName(lag1.Name()).SetRxName(port1.Name())
	flow1.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(80)
	flow1.Size().SetChoice("fixed").SetFixed(128)
	flow1.Rate().SetChoice("pps").SetPps(10)
	flow1Eth := flow1.Packet().Add().SetChoice("ethernet").Ethernet()
	flow1Eth.Dst().SetChoice("value")
	flow1Eth.Src().SetChoice("value").SetValue("00:22:01:00:00:01")
	flow1IP := flow1.Packet().Add().Ipv4()
	flow1IP.Dst().SetChoice("value").SetValue("11.1.1.2")
	flow1IP.Src().SetChoice("value").SetValue("21.1.1.2")
	flowTcp := flow1.Packet().Add().Tcp()
	flowTcp.DstPort().SetChoice("values").SetValues([]int32{
		5001,
		5002,
		5003,
		5004,
		5005,
		5006,
		5007,
		5008,
	})
	flowTcp.SrcPort().SetChoice("value").SetValue(4001)

	return config
}
