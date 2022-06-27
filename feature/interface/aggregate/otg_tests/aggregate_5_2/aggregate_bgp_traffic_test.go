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
	"github.com/openconfig/featureprofiles/tools/helpers"
	"github.com/openconfig/ondatra"
	"github.com/openconfig/ondatra/telemetry"
)

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

func getInterfaceStatus(t *testing.T, dut *ondatra.DUTDevice, interfaceName string) telemetry.E_Interface_OperStatus {
	status := dut.Telemetry().Interface(interfaceName).OperStatus().Get(t)
	t.Logf("Status of interface %s is %s\n", interfaceName, status)
	return status
}

func verifyPortStatus(t *testing.T, dut *ondatra.DUTDevice, interfaceName string, expStatus telemetry.E_Interface_OperStatus) (bool, error) {
	actStatus := getInterfaceStatus(t, dut, interfaceName)
	if actStatus != expStatus {
		return false, nil
	}
	return true, nil
}

func getLacpMembers(t *testing.T, dut *ondatra.DUTDevice, interfaceName string) []string {
	memberInterfaces := []string{}
	members := dut.Telemetry().Lacp().Interface(interfaceName).MemberAny().Get(t)
	for _, member := range members {
		memberInterfaces = append(memberInterfaces, member.GetInterface())
	}
	t.Logf("Bundled Ports for %s is : %v", interfaceName, memberInterfaces)
	return memberInterfaces
}

func bundledPortsAsExpected(t *testing.T, dut *ondatra.DUTDevice, expectedBundledPortsMap map[string][]string) (bool, error) {
	for iFace, expectedBundledPorts := range expectedBundledPortsMap {
		actualBundledPorts := getLacpMembers(t, dut, iFace)
		if !helpers.UnorderedEqual(expectedBundledPorts, actualBundledPorts) {
			return false, nil
		}
	}
	return true, nil
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
	err := helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_UP)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedBundledPortsMap := map[string][]string{
		"Port-Channel1": {
			dut.Port(t, "port2").Name(),
			dut.Port(t, "port3").Name(),
			dut.Port(t, "port4").Name(),
			dut.Port(t, "port5").Name(),
		},
	}

	fmt.Println(expectedBundledPortsMap)

	err = helpers.WaitFor(t, func() (bool, error) { return bundledPortsAsExpected(t, dut, expectedBundledPortsMap) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	// as up links >  min links
	makeMemberPortDown(t, dut, "port2")

	t.Logf("Check Interface status on DUT")
	err = helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_UP)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedBundledPortsMap["Port-Channel1"] = helpers.Remove(expectedBundledPortsMap["Port-Channel1"], dut.Port(t, "port2").Name())
	err = helpers.WaitFor(t, func() (bool, error) { return bundledPortsAsExpected(t, dut, expectedBundledPortsMap) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Keep BGP running for 20s After Tx port switch")
	time.Sleep(20 * time.Second)

	// as up links =  min links
	makeMemberPortDown(t, dut, "port3")

	t.Logf("Check Interface status on DUT")
	err = helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_UP)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedBundledPortsMap["Port-Channel1"] = helpers.Remove(expectedBundledPortsMap["Port-Channel1"], dut.Port(t, "port3").Name())
	err = helpers.WaitFor(t, func() (bool, error) { return bundledPortsAsExpected(t, dut, expectedBundledPortsMap) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	// as up links < min links
	makeMemberPortDown(t, dut, "port4")

	t.Logf("Check Interface status on DUT")
	err = helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_LOWER_LAYER_DOWN)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
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
	lag1port1.Ethernet().SetMac("00:00:00:00:00:17").SetName("lag1.port2.eth")

	// port4 as port of lag1
	lag1port3 := lag1.Ports().Add().
		SetPortName(port4.Name())
	lag1port3.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(3).
		SetActorPortPriority(1).
		SetLacpduTimeout(0)
	lag1port1.Ethernet().SetMac("00:00:00:00:00:18").SetName("lag1.port3.eth")

	// port5 as port of lag1
	lag1port4 := lag1.Ports().Add().
		SetPortName(port5.Name())
	lag1port4.Lacp().
		SetActorActivity("active").
		SetActorPortNumber(4).
		SetActorPortPriority(1).
		SetLacpduTimeout(0)
	lag1port1.Ethernet().SetMac("00:00:00:00:00:19").SetName("lag1.port4.eth")

	return config
}
