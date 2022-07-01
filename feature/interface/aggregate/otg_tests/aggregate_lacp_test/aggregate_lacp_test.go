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
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/featureprofiles/internal/fptest"
	"github.com/openconfig/featureprofiles/internal/otgutils"
	"github.com/openconfig/ondatra"
	"github.com/openconfig/ondatra/telemetry"
	otgtelemetry "github.com/openconfig/ondatra/telemetry/otg"
)

type DUTLacpMember struct {
	Collecting   bool
	Distributing bool
}

type OtgLagMetric struct {
	Status        string
	MemberPortsUp int32
}

type OtgLacpMetric struct {
	Collecting   bool
	Distributing bool
}

func indexInSlice(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 //not found.
}

func removeFromSlice(slice []string, s string) []string {
	i := indexInSlice(s, slice)
	return append(slice[:i], slice[i+1:]...)
}

func isUnorderedEqual(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	exists := make(map[string]bool)
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if !exists[value] {
			return false
		}
	}
	return true
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

func makeMemberPortDown(t *testing.T, dut *ondatra.DUTDevice, portId string) {
	t.Logf("Making port %s down for DUT...\n", portId)
	configText := fmt.Sprintf("interface %s\nno channel-group 1 mode active\n!", dut.Port(t, portId).Name())
	dut.Config().New().WithAristaText(configText).Append(t)
}

func dutVerifyInterfaceStatus(t *testing.T, dut *ondatra.DUTDevice, interfaceName string, expStatus string) {
	interfacePath := dut.Telemetry().Interface(interfaceName)
	_, ok := interfacePath.OperStatus().Watch(t, time.Minute,
		func(val *telemetry.QualifiedE_Interface_OperStatus) bool {
			return val.IsPresent() && val.Val(t).String() == expStatus
		}).Await(t)
	if !ok {
		t.Fatal(t, "Interface reported Oper status", interfacePath.OperStatus().Get(t))
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
		}
	}
	return true, nil
}

func getDutBundledInterfaces(t *testing.T, dut *ondatra.DUTDevice, interfaceName string) []string {
	memberInterfaces := []string{}
	members := dut.Telemetry().Lacp().Interface(interfaceName).MemberAny().Get(t)
	for _, member := range members {
		memberInterfaces = append(memberInterfaces, member.GetInterface())
	}
	t.Logf("Bundled Ports for %s is : %v", interfaceName, memberInterfaces)
	return memberInterfaces
}

func dutBundledInterfacesAsExpected(t *testing.T, dut *ondatra.DUTDevice, expectedBundledPortsMap map[string][]string) {
	for iFace, expectedBundledPorts := range expectedBundledPortsMap {
		actualBundledPorts := getDutBundledInterfaces(t, dut, iFace)
		if !isUnorderedEqual(expectedBundledPorts, actualBundledPorts) {
			t.Fatal("Bundled ports for ", iFace, " is ", actualBundledPorts, " expected: ", expectedBundledPorts)
		}
	}
}

func otgLagAsExpected(t *testing.T, otg *ondatra.OTG, expectedOtgLagMetrics map[string]OtgLagMetric) {
	for lag, expOtgLagMetric := range expectedOtgLagMetrics {
		lagPath := otg.Telemetry().Lag(lag)
		_, ok := lagPath.OperStatus().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedE_Lag_OperStatus) bool {
				return val.IsPresent() && val.Val(t).String() == expOtgLagMetric.Status
			}).Await(t)
		if !ok {
			t.Fatal(t, "for Lag ", lag, " Oper Status: ", lagPath.OperStatus().Get(t))
		}

		_, ok = lagPath.Counters().MemberPortsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == uint64(expOtgLagMetric.MemberPortsUp)
			}).Await(t)
		if !ok {
			t.Fatal(t, "For Lag ", lag, " Member Ports Up: ", lagPath.OperStatus().Get(t))
		}
	}
}

func otgLacpAsExpected(t *testing.T, otg *ondatra.OTG, expectedOtgLacpMetrics map[string]OtgLacpMetric) {
	for lacpMemberPort, expOtgLacpMetric := range expectedOtgLacpMetrics {
		lacpMemberPath := otg.Telemetry().Lacp().LagMember(lacpMemberPort)
		_, ok := lacpMemberPath.Collecting().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedBool) bool {
				return val.IsPresent() && val.Val(t) == expOtgLacpMetric.Collecting
			}).Await(t)
		if !ok {
			t.Fatal(t, "for Lacp Port ", lacpMemberPort, " Collecting is: ", lacpMemberPath.Collecting().Get(t))
		}

		_, ok = lacpMemberPath.Distributing().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedBool) bool {
				return val.IsPresent() && val.Val(t) == expOtgLacpMetric.Distributing
			}).Await(t)
		if !ok {
			t.Fatal(t, "for Lacp Port ", lacpMemberPort, " Distributing is: ", lacpMemberPath.Distributing().Get(t))
		}
	}
}

func TestAggregateBGPTraffic(t *testing.T) {
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
			dut.Port(t, "port2").Name(): {Collecting: true, Distributing: true},
			dut.Port(t, "port3").Name(): {Collecting: true, Distributing: true},
			dut.Port(t, "port4").Name(): {Collecting: true, Distributing: true},
			dut.Port(t, "port5").Name(): {Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check Lacp Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedBundledPortsMap := map[string][]string{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(),
			dut.Port(t, "port3").Name(),
			dut.Port(t, "port4").Name(),
			dut.Port(t, "port5").Name(),
		},
	}

	t.Logf("Checking bundled interfaces for given port channel on DUT")
	dutBundledInterfacesAsExpected(t, dut, expectedBundledPortsMap)

	expectedOtgLagMetrics := map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 4},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedOtgLacpMetrics := map[string]OtgLacpMetric{
		"port2": {
			Collecting:   true,
			Distributing: true,
		},
		"port3": {
			Collecting:   true,
			Distributing: true,
		},
		"port4": {
			Collecting:   true,
			Distributing: true,
		},
		"port5": {
			Collecting:   true,
			Distributing: true,
		},
	}

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	// as up links >  min links
	fmt.Println("Making Lag Member port2 down")
	otg.DownLacpMember(t, []string{"port2"})

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(): {Collecting: false, Distributing: false},
			dut.Port(t, "port3").Name(): {Collecting: true, Distributing: true},
			dut.Port(t, "port4").Name(): {Collecting: true, Distributing: true},
			dut.Port(t, "port5").Name(): {Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check Lacp Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 3},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
		"port2": {
			Collecting:   false,
			Distributing: false,
		},
		"port3": {
			Collecting:   true,
			Distributing: true,
		},
		"port4": {
			Collecting:   true,
			Distributing: true,
		},
		"port5": {
			Collecting:   true,
			Distributing: true,
		},
	}

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	makeMemberPortDown(t, dut, "port2")

	t.Logf("Check Interface status on DUT after bringing 1 of 4 port links down (up links > min links) ")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedBundledPortsMap["Port-Channel1"] = removeFromSlice(expectedBundledPortsMap["Port-Channel1"], dut.Port(t, "port2").Name())
	t.Logf("Checking bundled interfaces for given port channel on DUT")
	dutBundledInterfacesAsExpected(t, dut, expectedBundledPortsMap)

	// as up links =  min links
	fmt.Println("Making Lag Member port3 down")
	otg.DownLacpMember(t, []string{"port3"})

	expectedLacpMemberPortsMap = map[string]map[string]DUTLacpMember{
		"Port-Channel1": {
			dut.Port(t, "port3").Name(): {Collecting: false, Distributing: false},
			dut.Port(t, "port4").Name(): {Collecting: true, Distributing: true},
			dut.Port(t, "port5").Name(): {Collecting: true, Distributing: true},
		},
	}

	t.Logf("Check Lacp Member status on DUT")
	dutLacpMemberPortsAsExpected(t, dut, expectedLacpMemberPortsMap)

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "UP", MemberPortsUp: 2},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
		"port2": {
			Collecting:   false,
			Distributing: false,
		},
		"port3": {
			Collecting:   false,
			Distributing: false,
		},
		"port4": {
			Collecting:   true,
			Distributing: true,
		},
		"port5": {
			Collecting:   true,
			Distributing: true,
		},
	}

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	makeMemberPortDown(t, dut, "port3")

	t.Logf("Check Interface status on DUT after 2 of 4 port links down (up links = min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "UP")

	expectedBundledPortsMap["Port-Channel1"] = removeFromSlice(expectedBundledPortsMap["Port-Channel1"], dut.Port(t, "port3").Name())
	t.Logf("Checking bundled interfaces for given port channel on DUT")
	dutBundledInterfacesAsExpected(t, dut, expectedBundledPortsMap)

	// as up links < min links
	fmt.Println("Making Lag Member port4 down ")
	otg.DownLacpMember(t, []string{"port4"})

	expectedOtgLacpMetrics = map[string]OtgLacpMetric{
		"port2": {
			Collecting:   false,
			Distributing: false,
		},
		"port3": {
			Collecting:   false,
			Distributing: false,
		},
		"port4": {
			Collecting:   false,
			Distributing: false,
		},
		"port5": {
			Collecting:   true,
			Distributing: true,
		},
	}

	t.Logf("Checking Lacp metrics as expected on OTG")
	otgLacpAsExpected(t, otg, expectedOtgLacpMetrics)
	otgutils.LogLacpMetrics(t, otg, config)

	makeMemberPortDown(t, dut, "port4")

	expectedOtgLagMetrics = map[string]OtgLagMetric{
		"lag1": {Status: "DOWN", MemberPortsUp: 0},
	}

	t.Logf("Checking Lag metrics as expected on OTG")
	otgLagAsExpected(t, otg, expectedOtgLagMetrics)
	otgutils.LogLagMetrics(t, otg, config)

	t.Logf("Check Interface status on DUT after 3 of 4 port links down (up links < min links)")
	dutVerifyInterfaceStatus(t, dut, "Port-Channel1", "LOWER_LAYER_DOWN")
}

func configureOTG(t *testing.T, otg *ondatra.OTG) gosnappi.Config {
	config := otg.NewConfig(t)
	config.Ports().Add().SetName("port1")
	port2 := config.Ports().Add().SetName("port2")
	port3 := config.Ports().Add().SetName("port3")
	port4 := config.Ports().Add().SetName("port4")
	port5 := config.Ports().Add().SetName("port5")

	// lag1
	lag1 := config.Lags().Add().SetName("lag1")
	lag1.Protocol().Lacp().SetActorKey(1).SetActorSystemId("01:01:01:01:01:01").SetActorSystemPriority(1)

	// port2 as port of lag1
	lag1port1 := lag1.Ports().Add().
		SetPortName(port2.Name())
	lag1port1.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(1).
		SetActorPortPriority(1).
		SetLacpduTimeout(0)
	lag1port1.Ethernet().SetMac("00:00:00:00:00:16").SetName("lag1.port1.eth")

	// port3 as port of lag1
	lag1port2 := lag1.Ports().Add().
		SetPortName(port3.Name())
	lag1port2.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(2).
		SetActorPortPriority(1).
		SetLacpduTimeout(0)
	lag1port2.Ethernet().SetMac("00:00:00:00:00:17").SetName("lag1.port2.eth")

	// port4 as port of lag1
	lag1port3 := lag1.Ports().Add().
		SetPortName(port4.Name())
	lag1port3.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(3).
		SetActorPortPriority(1).
		SetLacpduTimeout(0)
	lag1port3.Ethernet().SetMac("00:00:00:00:00:18").SetName("lag1.port3.eth")

	// port5 as port of lag1
	lag1port4 := lag1.Ports().Add().
		SetPortName(port5.Name())
	lag1port4.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(4).
		SetActorPortPriority(1).
		SetLacpduTimeout(0)
	lag1port4.Ethernet().SetMac("00:00:00:00:00:19").SetName("lag1.port4.eth")

	return config
}