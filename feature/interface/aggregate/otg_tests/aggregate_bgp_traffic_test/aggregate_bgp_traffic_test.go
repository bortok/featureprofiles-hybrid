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

package rt_5_2_aggregate_bgp_traffic_test

import (
	"fmt"
	"testing"

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

func getInterfaceMacs(t *testing.T, dut *ondatra.DUTDevice) map[string]string {
	dutMacDetails := make(map[string]string)
	for _, p := range dut.Ports() {
		eth := dut.Telemetry().Interface(p.Name()).Ethernet().Get(t)
		t.Logf("Mac address of Interface %s in DUT: %s", p.Name(), eth.GetMacAddress())
		dutMacDetails[p.Name()] = eth.GetMacAddress()
	}
	return dutMacDetails
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
	config, expected := configureOTG(t, otg)
	dutMacDetails := getInterfaceMacs(t, dut)
	config.Flows().Items()[0].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[1].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[2].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])
	config.Flows().Items()[3].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacDetails[dut.Port(t, "port1").Name()])

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

	t.Logf("Check BGP sessions on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.Bgp4SessionAsExpected(t, otg, config, expected) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	otg.StartTraffic(t)

	t.Logf("Check Flow stats on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.PortAndFlowMetricsOk(t, otg, config) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	rPortId, err := helpers.GetFlowDestinationLocation(t, otg, config, 400)
	if err != nil {
		t.Fatal(err)
	}

	otg.StopTraffic(t)

	t.Logf("All packets are being received by port %s", rPortId)

	// as up links > min links
	makeMemberPortDown(t, dut, rPortId)

	t.Logf("Check Interface status on DUT")
	err = helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_UP)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedBundledPortsMap["Port-Channel1"] = helpers.Remove(expectedBundledPortsMap["Port-Channel1"], dut.Port(t, rPortId).Name())
	err = helpers.WaitFor(t, func() (bool, error) { return bundledPortsAsExpected(t, dut, expectedBundledPortsMap) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Check BGP sessions on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.Bgp4SessionAsExpected(t, otg, config, expected) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	otg.StartTraffic(t)

	t.Logf("Check Flow stats on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.PortAndFlowMetricsOk(t, otg, config) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	rPortId, err = helpers.GetFlowDestinationLocation(t, otg, config, 400)
	if err != nil {
		t.Fatal(err)
	}

	otg.StopTraffic(t)

	t.Logf("All packets are being received by port %s", rPortId)

	// as up links = min links
	makeMemberPortDown(t, dut, rPortId)

	t.Logf("Check Interface status on DUT")
	err = helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_UP)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedBundledPortsMap["Port-Channel1"] = helpers.Remove(expectedBundledPortsMap["Port-Channel1"], dut.Port(t, rPortId).Name())
	err = helpers.WaitFor(t, func() (bool, error) { return bundledPortsAsExpected(t, dut, expectedBundledPortsMap) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Check BGP sessions on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.Bgp4SessionAsExpected(t, otg, config, expected) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	otg.StartTraffic(t)

	t.Logf("Check Flow stats on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.PortAndFlowMetricsOk(t, otg, config) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	rPortId, err = helpers.GetFlowDestinationLocation(t, otg, config, 400)
	if err != nil {
		t.Fatal(err)
	}

	otg.StopTraffic(t)

	t.Logf("All packets are being received by port %s", rPortId)

	// as up links < min links
	makeMemberPortDown(t, dut, rPortId)

	t.Logf("Check Interface status on DUT")
	err = helpers.WaitFor(t, func() (bool, error) {
		return verifyPortStatus(t, dut, "Port-Channel1", telemetry.Interface_OperStatus_LOWER_LAYER_DOWN)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	expected = helpers.ExpectedState{
		Bgp4: map[string]helpers.ExpectedBgpMetrics{
			"p1d1.bgp1":   {State: gosnappi.Bgpv4MetricSessionState.UP, Advertised: 1, Received: 10},
			"lag1d1.bgp1": {State: gosnappi.Bgpv4MetricSessionState.DOWN, Advertised: 10, Received: 1},
		},
	}

	t.Logf("Check BGP sessions on OTG")
	err = helpers.WaitFor(t, func() (bool, error) { return helpers.Bgp4SessionAsExpected(t, otg, config, expected) }, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func configureOTG(t *testing.T, otg *ondatra.OTG) (gosnappi.Config, helpers.ExpectedState) {
	config := otg.NewConfig(t)
	port1 := config.Ports().Add().SetName("port1")
	port2 := config.Ports().Add().SetName("port2")
	port3 := config.Ports().Add().SetName("port3")
	port4 := config.Ports().Add().SetName("port4")
	port5 := config.Ports().Add().SetName("port5")

	// lag1
	lag1 := config.Lags().Add().SetName("lag1")

	// port2 as port of lag1
	lag1port1 := lag1.Ports().Add().SetPortName(port2.Name())
	lag1port1.Protocol().SetChoice("lacp").Lacp().
		SetActorKey(1).
		SetActorPortNumber(1).
		SetActorPortPriority(1).
		SetActorSystemId("01:01:01:01:01:01").
		SetActorSystemPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(1).
		SetActorActivity("active")

	lag1port1.Ethernet().SetName("lag1.port1.eth").
		SetMac("00:00:00:00:00:16")

	// port3 as port of lag1
	lag1port2 := lag1.Ports().Add().SetPortName(port3.Name())
	lag1port2.Protocol().SetChoice("lacp").Lacp().
		SetActorKey(1).
		SetActorPortNumber(2).
		SetActorPortPriority(1).
		SetActorSystemId("01:01:01:01:01:01").
		SetActorSystemPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(1).
		SetActorActivity("active")

	lag1port2.Ethernet().SetName("lag1.port2.eth").
		SetMac("00:00:00:00:00:17")

	// port4 as port of lag1
	lag1port3 := lag1.Ports().Add().SetPortName(port4.Name())
	lag1port3.Protocol().SetChoice("lacp").Lacp().
		SetActorKey(1).
		SetActorPortNumber(3).
		SetActorPortPriority(1).
		SetActorSystemId("01:01:01:01:01:01").
		SetActorSystemPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(1).
		SetActorActivity("active")

	lag1port3.Ethernet().SetName("lag1.port3.eth").
		SetMac("00:00:00:00:00:18")

	// port5 as port of lag1
	lag1port4 := lag1.Ports().Add().SetPortName(port5.Name())
	lag1port4.Protocol().SetChoice("lacp").Lacp().
		SetActorKey(1).
		SetActorPortNumber(4).
		SetActorPortPriority(1).
		SetActorSystemId("01:01:01:01:01:01").
		SetActorSystemPriority(1).
		SetLacpduPeriodicTimeInterval(1).
		SetLacpduTimeout(1).
		SetActorActivity("active")

	lag1port4.Ethernet().SetName("lag1.port4.eth").
		SetMac("00:00:00:00:00:19")

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

	p1d1inf := p1d1.Bgp().SetRouterId("11.1.1.2").Ipv4Interfaces().Add().
		SetIpv4Name(p1d1eth1ip1.Name())

	p1d1infpeer1 := p1d1inf.Peers().Add().
		SetName("p1d1.bgp1").
		SetPeerAddress("11.1.1.1").
		SetAsNumber(65100).
		SetAsType("ebgp")

	p1d1infpeer1.Advanced().SetKeepAliveInterval(5)

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

	lag1d1inf := lag1d1.Bgp().SetRouterId("21.1.1.2").Ipv4Interfaces().Add().
		SetIpv4Name(lag1d1eth1ip1.Name())

	lag1d1infpeer1 := lag1d1inf.Peers().Add().
		SetName("lag1d1.bgp1").
		SetPeerAddress("21.1.1.1").
		SetAsNumber(65300).
		SetAsType("ebgp")

	p1d1infpeer1.Advanced().SetKeepAliveInterval(5)

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

	// flow port1 -> port2
	flow1 := config.Flows().Add().SetName("port1->port2")
	flow1.Metrics().SetEnable(true)
	flow1.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port2.Name())
	flow1.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(100)
	flow1.Size().SetChoice("fixed").SetFixed(128)
	flow1.Rate().SetChoice("pps").SetPps(100)
	flow1Eth := flow1.Packet().Add().SetChoice("ethernet").Ethernet()
	flow1Eth.Dst().SetChoice("value")
	flow1Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow1IP := flow1.Packet().Add().Ipv4()
	flow1IP.Dst().SetChoice("increment").Increment().SetStart("211.1.0.1").SetStep("0.0.0.1").SetCount(1)
	flow1IP.Src().SetChoice("increment").Increment().SetStart("111.1.0.1").SetStep("0.0.0.1").SetCount(1)
	flow1Tcp := flow1.Packet().Add().Tcp()
	flow1Tcp.SrcPort().SetChoice("value").SetValue(5001)
	flow1Tcp.DstPort().SetChoice("value").SetValue(5002)
	flow1Tcp.Window().SetChoice("value").SetValue(1)

	// flow port1 -> port3
	flow2 := config.Flows().Add().SetName("port1->port3")
	flow2.Metrics().SetEnable(true)
	flow2.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port3.Name())
	flow2.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(100)
	flow2.Size().SetChoice("fixed").SetFixed(128)
	flow2.Rate().SetChoice("pps").SetPps(100)
	flow2Eth := flow2.Packet().Add().SetChoice("ethernet").Ethernet()
	flow2Eth.Dst().SetChoice("value")
	flow2Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow2IP := flow2.Packet().Add().Ipv4()
	flow2IP.Dst().SetChoice("increment").Increment().SetStart("211.1.0.2").SetStep("0.0.0.1").SetCount(1)
	flow2IP.Src().SetChoice("increment").Increment().SetStart("111.1.0.1").SetStep("0.0.0.1").SetCount(1)
	flow2Tcp := flow2.Packet().Add().Tcp()
	flow2Tcp.SrcPort().SetChoice("value").SetValue(5001)
	flow2Tcp.DstPort().SetChoice("value").SetValue(5002)
	flow2Tcp.Window().SetChoice("value").SetValue(1)

	// flow port1 -> port4
	flow3 := config.Flows().Add().SetName("port1->port4")
	flow3.Metrics().SetEnable(true)
	flow3.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port4.Name())
	flow3.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(100)
	flow3.Size().SetChoice("fixed").SetFixed(128)
	flow3.Rate().SetChoice("pps").SetPps(100)
	flow3Eth := flow3.Packet().Add().SetChoice("ethernet").Ethernet()
	flow3Eth.Dst().SetChoice("value")
	flow3Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow3IP := flow3.Packet().Add().Ipv4()
	flow3IP.Dst().SetChoice("increment").Increment().SetStart("211.1.0.3").SetStep("0.0.0.1").SetCount(1)
	flow3IP.Src().SetChoice("increment").Increment().SetStart("111.1.0.1").SetStep("0.0.0.1").SetCount(1)
	flow3Tcp := flow3.Packet().Add().Tcp()
	flow3Tcp.SrcPort().SetChoice("value").SetValue(5001)
	flow3Tcp.DstPort().SetChoice("value").SetValue(5002)
	flow3Tcp.Window().SetChoice("value").SetValue(1)

	// flow port1 -> port5
	flow4 := config.Flows().Add().SetName("port1->port5")
	flow4.Metrics().SetEnable(true)
	flow4.TxRx().SetChoice("port").Port().SetTxName(port1.Name()).SetRxName(port5.Name())
	flow4.Duration().SetChoice("fixed_packets").FixedPackets().SetPackets(100)
	flow4.Size().SetChoice("fixed").SetFixed(128)
	flow4.Rate().SetChoice("pps").SetPps(100)
	flow4Eth := flow4.Packet().Add().SetChoice("ethernet").Ethernet()
	flow4Eth.Dst().SetChoice("value")
	flow4Eth.Src().SetChoice("value").SetValue("00:00:01:01:01:01")
	flow4IP := flow4.Packet().Add().Ipv4()
	flow4IP.Dst().SetChoice("increment").Increment().SetStart("211.1.0.4").SetStep("0.0.0.1").SetCount(1)
	flow4IP.Src().SetChoice("increment").Increment().SetStart("111.1.0.1").SetStep("0.0.0.1").SetCount(1)
	flow4Tcp := flow4.Packet().Add().Tcp()
	flow4Tcp.SrcPort().SetChoice("value").SetValue(5001)
	flow4Tcp.DstPort().SetChoice("value").SetValue(5002)
	flow4Tcp.Window().SetChoice("value").SetValue(1)

	expected := helpers.ExpectedState{
		Bgp4: map[string]helpers.ExpectedBgpMetrics{
			p1d1infpeer1.Name():   {State: gosnappi.Bgpv4MetricSessionState.UP, Advertised: 1, Received: 10},
			lag1d1infpeer1.Name(): {State: gosnappi.Bgpv4MetricSessionState.UP, Advertised: 10, Received: 1},
		},
	}

	return config, expected
}
