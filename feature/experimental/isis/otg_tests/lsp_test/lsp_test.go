package aggregate_traffic

import (
	"reflect"
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/featureprofiles/internal/fptest"
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
	Ipv6ReachabilityTlvCount         int
}

type OtgISISMetric struct {
	L1SessionsUp   uint64
	L1SessionsFlap uint64
	L1DatabaseSize uint64
	L2SessionsUp   uint64
	L2SessionsFlap uint64
	L2DatabaseSize uint64
}

func checkOTGISISLspStates(t *testing.T, otg *otg.OTG, config gosnappi.Config, expectedOTGISISLspStates map[string][]OtgISISLsp) bool {
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
								if reflect.DeepEqual(isisLspTlvs.GetHostnames().GetHostname(), otgISISLsp.Hostnames) {
									if isisLspTlvs.ExtendedIpv4Reachability != nil {
										if len(isisLspTlvs.GetExtendedIpv4Reachability().Prefix) == otgISISLsp.ExtendedIpv4ReachabilityTlvCount {
											if isisLspTlvs.Ipv6Reachability != nil {
												if len(isisLspTlvs.GetIpv6Reachability().Prefix) == otgISISLsp.Ipv6ReachabilityTlvCount {
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
