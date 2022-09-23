package isis_lsp

import (
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/featureprofiles/internal/fptest"
	"github.com/openconfig/featureprofiles/internal/otgutils"
	"github.com/openconfig/ondatra"
	"github.com/openconfig/ondatra/otg"
	otgtelemetry "github.com/openconfig/ondatra/telemetry/otg"
)

type OtgISISLsp struct {
	LspId                            string
	PduType                          otgtelemetry.E_Lsps_PduType
	ISType                           uint8
	Hostnames                        []string
	ExtendedIsReachabilityTlvCount   int
	ExtendedIpv4ReachabilityTlvCount int
}

type OtgISISMetric struct {
	L1SessionsUp   uint64
	L1SessionsFlap uint64
	L1DatabaseSize uint64
	L2SessionsUp   uint64
	L2SessionsFlap uint64
	L2DatabaseSize uint64
}

type OtgFlowMetric struct {
	TxPackets uint64
	RxPackets uint64
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func checkOTGISISLspStates(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedOTGISISLspStates map[string][]OtgISISLsp) bool {
	otgutils.LogISISLspStates(t, otg, config)
	for routerName, otgISISLsps := range expectedOTGISISLspStates {
		isisLsps := otg.Telemetry().IsisRouter(routerName).LinkStateDatabase().LspsAny().Get(t)
		for _, otgISISLsp := range otgISISLsps {
			found := false
			for _, isisLsp := range isisLsps {
				if isisLsp.GetLspId() == otgISISLsp.LspId {
					if isisLsp.GetPduType() == otgISISLsp.PduType {
						if isisLsp.GetIsType() == otgISISLsp.ISType {
							if isisLsp.Tlvs != nil {
								isisLspTlvs := isisLsp.GetTlvs()
								if sliceEqual(isisLspTlvs.GetHostnames().GetHostname(), otgISISLsp.Hostnames) {
									if isisLspTlvs.ExtendedIpv4Reachability != nil {
										if len(isisLspTlvs.GetExtendedIpv4Reachability().Prefix) == otgISISLsp.ExtendedIpv4ReachabilityTlvCount {
											if isisLspTlvs.ExtendedIsReachability != nil {
												if len(isisLspTlvs.GetExtendedIsReachability().Neighbor) == otgISISLsp.ExtendedIsReachabilityTlvCount {
													found = true
													break
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func waitFor(fn func() bool, t testing.TB) {
	start := time.Now()
	for {
		done := fn()
		if done {
			t.Logf("Expected BGP Prefix received")
			break
		}
		if time.Since(start) > time.Minute {
			t.Errorf("Timeout while waiting for expected stats...")
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func verifyOTGISISTelemetry(t *testing.T, otg *otg.OTG, c gosnappi.Config, expectedOTGISISMetric map[string]OtgISISMetric) {
	for routerName, expectedOTGISISMetric := range expectedOTGISISMetric {
		isisCounterPath := otg.Telemetry().IsisRouter(routerName).Counters()
		_, ok := isisCounterPath.Level1().SessionsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedOTGISISMetric.L1SessionsUp
			}).Await(t)
		if !ok {
			t.Fatal(t, "ISIS router ", routerName, " L1SessionUp is ", isisCounterPath.Level1().SessionsUp().Get(t), " but Expected ", expectedOTGISISMetric.L1SessionsUp)
		}
		_, ok = isisCounterPath.Level1().SessionsFlap().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedOTGISISMetric.L1SessionsFlap
			}).Await(t)
		if !ok {
			t.Fatal(t, "ISIS router ", routerName, " L1SessionsFlap is ", isisCounterPath.Level1().SessionsFlap().Get(t), " but Expected ", expectedOTGISISMetric.L1SessionsFlap)
		}

		_, ok = isisCounterPath.Level1().DatabaseSize().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedOTGISISMetric.L1DatabaseSize
			}).Await(t)
		if !ok {
			t.Fatal(t, "ISIS router ", routerName, " L1DatabaseSize is ", isisCounterPath.Level1().DatabaseSize().Get(t), " but Expected ", expectedOTGISISMetric.L1DatabaseSize)
		}

		_, ok = isisCounterPath.Level2().SessionsUp().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedOTGISISMetric.L2SessionsUp
			}).Await(t)
		if !ok {
			t.Fatal(t, "ISIS router ", routerName, " L2SessionUp is ", isisCounterPath.Level2().SessionsUp().Get(t), " but Expected ", expectedOTGISISMetric.L2SessionsUp)
		}
		_, ok = isisCounterPath.Level2().SessionsFlap().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedOTGISISMetric.L2SessionsFlap
			}).Await(t)
		if !ok {
			t.Fatal(t, "ISIS router ", routerName, " L2SessionsFlap is ", isisCounterPath.Level2().SessionsFlap().Get(t), " but Expected ", expectedOTGISISMetric.L2SessionsFlap)
		}

		_, ok = isisCounterPath.Level2().DatabaseSize().Watch(t, time.Minute,
			func(val *otgtelemetry.QualifiedUint64) bool {
				return val.IsPresent() && val.Val(t) == expectedOTGISISMetric.L2DatabaseSize
			}).Await(t)
		if !ok {
			t.Fatal(t, "ISIS router ", routerName, " L2DatabaseSize is ", isisCounterPath.Level2().DatabaseSize().Get(t), " but Expected ", expectedOTGISISMetric.L2DatabaseSize)
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

func TestMain(m *testing.M) {
	fptest.RunTests(m)
}

func configureDUT(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Configuring DUT...")
	if dut.Port(t, "port1").Name() == "Ethernet1" {
		dut.Config().New().WithAristaFile("set_arista.config").Push(t)
	} else {
		dut.Config().New().WithAristaFile("set_arista_alt.config").Push(t)
	}

}

func unsetDUT(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Configuring DUT...")
	dut.Config().New().WithAristaFile("unset_arista.config").Push(t)
}

func TestIsisLSP(t *testing.T) {
	dut := ondatra.DUT(t, "dut")
	configureDUT(t, dut)
	defer unsetDUT(t, dut)

	ate := ondatra.ATE(t, "ate")
	otg := ate.OTG()
	config := configureOTG(t, otg)

	otg.PushConfig(t, config)
	otg.StartProtocols(t)
	defer otg.StopProtocols(t)

	expectedISISMetric := map[string]OtgISISMetric{
		"p1d1Isis": {
			L1SessionsUp:   0,
			L1SessionsFlap: 0,
			L1DatabaseSize: 0,
			L2SessionsUp:   1,
			L2SessionsFlap: 0,
			L2DatabaseSize: 3,
		},
		"p2d1Isis": {
			L1SessionsUp:   0,
			L1SessionsFlap: 0,
			L1DatabaseSize: 0,
			L2SessionsUp:   1,
			L2SessionsFlap: 0,
			L2DatabaseSize: 3,
		},
	}
	t.Logf("Verify ISIS session metrics...")
	verifyOTGISISTelemetry(t, otg, config, expectedISISMetric)
	otgutils.LogISISMetrics(t, otg, config)

	expectedISISLspStates := map[string][]OtgISISLsp{
		"p1d1Isis": {
			{
				LspId:                            "101010501040-00-00",
				PduType:                          otgtelemetry.Lsps_PduType_LEVEL_2,
				ISType:                           3,
				Hostnames:                        []string{},
				ExtendedIsReachabilityTlvCount:   2,
				ExtendedIpv4ReachabilityTlvCount: 2,
			},
			{
				LspId:                            "650000000001-00-00",
				PduType:                          otgtelemetry.Lsps_PduType_LEVEL_2,
				ISType:                           3,
				Hostnames:                        []string{"ixia-c-port2"},
				ExtendedIsReachabilityTlvCount:   1,
				ExtendedIpv4ReachabilityTlvCount: 4,
			},
		},
		"p2d1Isis": {
			{
				LspId:                            "101010501040-00-00",
				PduType:                          otgtelemetry.Lsps_PduType_LEVEL_2,
				ISType:                           3,
				Hostnames:                        []string{},
				ExtendedIsReachabilityTlvCount:   2,
				ExtendedIpv4ReachabilityTlvCount: 2,
			},
			{
				LspId:                            "640000000001-00-00",
				PduType:                          otgtelemetry.Lsps_PduType_LEVEL_2,
				ISType:                           3,
				Hostnames:                        []string{"ixia-c-port1"},
				ExtendedIsReachabilityTlvCount:   1,
				ExtendedIpv4ReachabilityTlvCount: 4,
			},
		},
	}
	t.Logf("Verify ISIS LSP states...")
	waitFor(func() bool { return checkOTGISISLspStates(t, otg, config, expectedISISLspStates) }, t)

	otg.StartTraffic(t)
	defer otg.StopTraffic(t)

	expectedOtgFlowMetrics := map[string]OtgFlowMetric{
		"p1-2.vlan.100": {
			TxPackets: 1000,
			RxPackets: 1000,
		},
		"p2-1.vlan.200": {
			TxPackets: 1000,
			RxPackets: 1000,
		},
	}
	t.Logf("Verify Flow metrics...")
	otgFlowMetricAsExpected(t, otg, config, expectedOtgFlowMetrics)
	otgutils.LogPortMetrics(t, otg, config)
	otgutils.LogFlowMetrics(t, otg, config)
}

func configureOTG(t *testing.T, otg *otg.OTG) gosnappi.Config {
	config := otg.NewConfig(t)
	port1 := config.Ports().Add().SetName("port1")
	port2 := config.Ports().Add().SetName("port2")

	// port 1 device 1
	p1d1 := config.Devices().Add().SetName("p1d1")
	// port 1 device 1 ethernet
	p1d1Eth := p1d1.Ethernets().Add().
		SetName("p1d1Eth").
		SetMac("00:00:01:01:01:01").
		SetMtu(1500).
		SetPortName(port1.Name())

	// port 1 device 1 ipv4
	p1d1Ipv4 := p1d1Eth.Ipv4Addresses().
		Add().
		SetAddress("1.1.1.2").
		SetGateway("1.1.1.1").
		SetName("p1d1Ipv4").
		SetPrefix(24)

	// port 1 device 1 vlan
	p1d1Vlan := p1d1Eth.Vlans().Add().
		SetId(100).
		SetName("p1d1vlan")

	// port 1 device 1 isis
	p1d1Isis := p1d1.Isis().SetName("p1d1Isis").SetSystemId("640000000001")

	// port 1 device 1 isis basic
	p1d1Isis.Basic().SetIpv4TeRouterId(p1d1Ipv4.Address())
	p1d1Isis.Basic().SetHostname("ixia-c-port1")
	p1d1Isis.Basic().SetEnableWideMetric(true)
	p1d1Isis.Basic().SetLearnedLspFilter(true)

	// port 1 device 1 isis advance
	p1d1Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p1d1Isis.Advanced().SetCsnpInterval(10000)
	p1d1Isis.Advanced().SetEnableHelloPadding(true)
	p1d1Isis.Advanced().SetLspLifetime(1200)
	p1d1Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d1Isis.Advanced().SetLspRefreshRate(900)
	p1d1Isis.Advanced().SetMaxAreaAddresses(3)
	p1d1Isis.Advanced().SetMaxLspSize(1492)
	p1d1Isis.Advanced().SetPsnpInterval(2000)
	p1d1Isis.Advanced().SetEnableAttachedBit(false)

	// port 1 device 1 isis interface
	p1d1IsisIntf := p1d1Isis.Interfaces().Add().
		SetEthName(p1d1Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p1d1IsisIntf")
	p1d1IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p1d1IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 1 device 1 isis v4 routes
	p1d1Isisv4routes := p1d1Isis.
		V4Routes().
		Add().
		SetName("p1d1IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p1d1Isisv4routes.Addresses().Add().
		SetAddress("10.10.1.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 2 device 1
	p2d1 := config.Devices().Add().SetName("p2d1")
	// port 2 device 1 ethernet
	p2d1Eth := p2d1.Ethernets().Add().
		SetName("p2d1Eth").
		SetMac("00:00:02:02:02:02").
		SetMtu(1500).
		SetPortName(port2.Name())

	// port 2 device 1 ipv4
	p2d1Ipv4 := p2d1Eth.Ipv4Addresses().
		Add().
		SetAddress("2.2.1.2").
		SetGateway("2.2.1.1").
		SetName("p2d1Ipv4").
		SetPrefix(24)

	// port 2 device 1 vlan
	p2d1Vlan := p2d1Eth.Vlans().Add().
		SetId(200).
		SetName("p2d1vlan")

	// port 2 device 1 isis
	p2d1Isis := p2d1.Isis().SetName("p2d1Isis").SetSystemId("650000000001")

	// port 2 device 1 isis basic
	p2d1Isis.Basic().SetIpv4TeRouterId(p2d1Ipv4.Address())
	p2d1Isis.Basic().SetHostname("ixia-c-port2")
	p2d1Isis.Basic().SetEnableWideMetric(true)
	p2d1Isis.Basic().SetLearnedLspFilter(true)

	// port 2 device 1 isis advance
	p2d1Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p2d1Isis.Advanced().SetCsnpInterval(10000)
	p2d1Isis.Advanced().SetEnableHelloPadding(true)
	p2d1Isis.Advanced().SetLspLifetime(1200)
	p2d1Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p2d1Isis.Advanced().SetLspRefreshRate(900)
	p2d1Isis.Advanced().SetMaxAreaAddresses(3)
	p2d1Isis.Advanced().SetMaxLspSize(1492)
	p2d1Isis.Advanced().SetPsnpInterval(2000)
	p2d1Isis.Advanced().SetEnableAttachedBit(false)

	// port 2 device 1 isis interface
	p2d1IsisIntf := p2d1Isis.Interfaces().Add().
		SetEthName(p2d1Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p2d1IsisIntf")
	p2d1IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p2d1IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 2 device 1 isis v4 routes
	p2d1Isisv4routes := p2d1Isis.
		V4Routes().
		Add().
		SetName("p2d1IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p2d1Isisv4routes.Addresses().Add().
		SetAddress("20.20.1.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// OTG traffic configuration
	f1 := config.Flows().Add().SetName("p1-2.vlan.100")
	f1.Metrics().SetEnable(true)
	f1.TxRx().Device().
		SetTxNames([]string{p1d1Isisv4routes.Name()}).
		SetRxNames([]string{p2d1Isisv4routes.Name()})
	f1.Size().SetFixed(512)
	f1.Rate().SetPps(500)
	f1.Duration().FixedPackets().SetPackets(1000)
	e1 := f1.Packet().Add().Ethernet()
	e1.Src().SetValue(p1d1Eth.Mac())

	vlan1 := f1.Packet().Add().Vlan()
	vlan1.Id().SetValue(p1d1Vlan.Id())
	vlan1.Tpid().SetValue(33024)

	v4 := f1.Packet().Add().Ipv4()
	v4.Src().SetValue("10.10.1.1")
	v4.Dst().SetValue("20.20.1.1")

	f2 := config.Flows().Add().SetName("p2-1.vlan.200")
	f2.Metrics().SetEnable(true)
	f2.TxRx().Device().
		SetTxNames([]string{p2d1Isisv4routes.Name()}).
		SetRxNames([]string{p1d1Isisv4routes.Name()})
	f2.Size().SetFixed(512)
	f2.Rate().SetPps(500)
	f2.Duration().FixedPackets().SetPackets(1000)
	e2 := f2.Packet().Add().Ethernet()
	e2.Src().SetValue(p2d1Eth.Mac())

	vlan2 := f2.Packet().Add().Vlan()
	vlan2.Id().SetValue(p2d1Vlan.Id())
	vlan2.Tpid().SetValue(33024)

	v4 = f2.Packet().Add().Ipv4()
	v4.Src().SetValue("20.20.1.1")
	v4.Dst().SetValue("10.10.1.1")

	return config
}
