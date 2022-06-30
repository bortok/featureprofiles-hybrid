package helpers

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
	otgtelemetry "github.com/openconfig/ondatra/telemetry/otg"
	"github.com/openconfig/ygot/ygot"
)

func GetFlowMetrics(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponseFlowMetricIter, error) {
	defer Timer(time.Now(), "GetFlowMetrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().FlowMetrics()
	for _, f := range c.Flows().Items() {
		log.Printf("Getting flow metrics for flow %s\n", f.Name())
		fMetric := metrics.Add()
		recvMetric := otg.Telemetry().Flow(f.Name()).Get(t)
		fMetric.SetName(recvMetric.GetName())
		fMetric.SetFramesRx(int64(recvMetric.GetCounters().GetInPkts()))
		fMetric.SetFramesTx(int64(recvMetric.GetCounters().GetOutPkts()))
		fMetric.SetFramesTxRate(ygot.BinaryToFloat32(recvMetric.GetOutFrameRate()))
		fMetric.SetFramesRxRate(ygot.BinaryToFloat32(recvMetric.GetInFrameRate()))
	}
	return metrics, nil
}

func GetPortMetrics(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponsePortMetricIter, error) {
	defer Timer(time.Now(), "GetPortMetrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().PortMetrics()
	for _, p := range c.Ports().Items() {
		log.Printf("Getting port metrics for port %s\n", p.Name())
		pMetric := metrics.Add()
		recvMetric := otg.Telemetry().Port(p.Name()).Get(t)
		pMetric.SetName(recvMetric.GetName())
		pMetric.SetFramesTx(int64(recvMetric.GetCounters().GetOutFrames()))
		pMetric.SetFramesRx(int64(recvMetric.GetCounters().GetInFrames()))
		pMetric.SetFramesTxRate(ygot.BinaryToFloat32(recvMetric.GetOutRate()))
	}
	return metrics, nil
}

func GetAllPortMetrics(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponsePortMetricIter, error) {
	defer Timer(time.Now(), "GetPortMetrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().PortMetrics()
	for _, p := range c.Ports().Items() {
		log.Printf("Getting port metrics for port %s\n", p.Name())
		pMetric := metrics.Add()
		recvMetric := otg.Telemetry().Port(p.Name()).Get(t)
		pMetric.SetName(recvMetric.GetName())
		pMetric.SetFramesTx(int64(recvMetric.GetCounters().GetOutFrames()))
		pMetric.SetFramesRx(int64(recvMetric.GetCounters().GetInFrames()))
		pMetric.SetBytesTx(int64(recvMetric.GetCounters().GetOutOctets()))
		pMetric.SetBytesRx(int64(recvMetric.GetCounters().GetInOctets()))
		pMetric.SetFramesTxRate(ygot.BinaryToFloat32(recvMetric.GetOutRate()))
		pMetric.SetFramesRxRate(ygot.BinaryToFloat32(recvMetric.GetInRate()))
		link := recvMetric.GetLink()
		if link == otgtelemetry.Port_Link_UP {
			pMetric.SetLink("up")
		} else {
			pMetric.SetLink("down")
		}

	}
	return metrics, nil
}

func GetBgpv4Metrics(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponseBgpv4MetricIter, error) {
	defer Timer(time.Now(), "GetBgpv4Metrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().Bgpv4Metrics()
	for _, d := range c.Devices().Items() {
		bgp := d.Bgp()
		for _, ip := range bgp.Ipv4Interfaces().Items() {
			for _, peer := range ip.Peers().Items() {
				log.Printf("Getting bgpv4 metrics for peer %s\n", peer.Name())
				bgpv4Metric := metrics.Add()
				recvMetricState := otg.Telemetry().BgpPeer(peer.Name()).SessionState().Get(t)
				recvMetricCounter := otg.Telemetry().BgpPeer(peer.Name()).Counters().Get(t)
				bgpv4Metric.SetName(peer.Name())
				bgpv4Metric.SetSessionFlapCount(int32(recvMetricCounter.GetFlaps()))
				bgpv4Metric.SetRoutesAdvertised(int32(recvMetricCounter.GetOutRoutes()))
				bgpv4Metric.SetRoutesReceived(int32(recvMetricCounter.GetInRoutes()))
				bgpv4Metric.SetRouteWithdrawsSent(int32(recvMetricCounter.GetOutRouteWithdraw()))
				bgpv4Metric.SetRouteWithdrawsReceived(int32(recvMetricCounter.GetInRouteWithdraw()))
				bgpv4Metric.SetKeepalivesSent(int32(recvMetricCounter.GetOutKeepalives()))
				bgpv4Metric.SetKeepalivesReceived(int32(recvMetricCounter.GetInKeepalives()))
				if recvMetricState == otgtelemetry.BgpPeer_SessionState_ESTABLISHED {
					bgpv4Metric.SetSessionState("up")
				} else {
					bgpv4Metric.SetSessionState("down")
				}
			}
		}
	}
	return metrics, nil
}

func GetBgpv6Metrics(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponseBgpv6MetricIter, error) {
	defer Timer(time.Now(), "GetBgpv6Metrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().Bgpv6Metrics()
	for _, d := range c.Devices().Items() {
		bgp := d.Bgp()
		for _, ipv6 := range bgp.Ipv6Interfaces().Items() {
			for _, peer := range ipv6.Peers().Items() {
				log.Printf("Getting bgpv6 metrics for peer %s\n", peer.Name())
				bgpv6Metric := metrics.Add()
				recvMetricState := otg.Telemetry().BgpPeer(peer.Name()).SessionState().Get(t)
				recvMetricCounter := otg.Telemetry().BgpPeer(peer.Name()).Counters().Get(t)
				bgpv6Metric.SetName(peer.Name())
				bgpv6Metric.SetSessionFlapCount(int32(recvMetricCounter.GetFlaps()))
				bgpv6Metric.SetRoutesAdvertised(int32(recvMetricCounter.GetOutRoutes()))
				bgpv6Metric.SetRoutesReceived(int32(recvMetricCounter.GetInRoutes()))
				bgpv6Metric.SetRouteWithdrawsSent(int32(recvMetricCounter.GetOutRouteWithdraw()))
				bgpv6Metric.SetRouteWithdrawsReceived(int32(recvMetricCounter.GetInRouteWithdraw()))
				bgpv6Metric.SetKeepalivesSent(int32(recvMetricCounter.GetOutKeepalives()))
				bgpv6Metric.SetKeepalivesReceived(int32(recvMetricCounter.GetInKeepalives()))
				if recvMetricState == otgtelemetry.BgpPeer_SessionState_ESTABLISHED {
					bgpv6Metric.SetSessionState("up")
				} else {
					bgpv6Metric.SetSessionState("down")
				}
			}
		}
	}
	return metrics, nil
}

func GetIsisMetrics(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponseIsisMetricIter, error) {
	defer Timer(time.Now(), "GetIsisMetrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().IsisMetrics()
	for _, d := range c.Devices().Items() {
		isis := d.Isis()
		log.Printf("Getting isis metrics for router %s\n", isis.Name())
		isisMetric := metrics.Add()
		recvMetric := otg.Telemetry().IsisRouter(isis.Name()).Get(t)
		isisMetric.SetName(recvMetric.GetName())
		isisMetric.SetL1SessionsUp(int32(recvMetric.GetCounters().GetLevel1().GetSessionsUp()))
		isisMetric.SetL1SessionFlap(int32(recvMetric.GetCounters().GetLevel1().GetSessionsFlap()))
		isisMetric.SetL1BroadcastHellosSent(int32(recvMetric.GetCounters().GetLevel1().GetOutBcastHellos()))
		isisMetric.SetL1BroadcastHellosReceived(int32(recvMetric.GetCounters().GetLevel1().GetInBcastHellos()))
		isisMetric.SetL1PointToPointHellosSent(int32(recvMetric.GetCounters().GetLevel1().GetOutP2PHellos()))
		isisMetric.SetL1PointToPointHellosReceived(int32(recvMetric.GetCounters().GetLevel1().GetInP2PHellos()))
		isisMetric.SetL1LspSent(int32(recvMetric.GetCounters().GetLevel1().GetOutLsp()))
		isisMetric.SetL1LspReceived(int32(recvMetric.GetCounters().GetLevel1().GetInLsp()))
		isisMetric.SetL1DatabaseSize(int32(recvMetric.GetCounters().GetLevel1().GetDatabaseSize()))
		isisMetric.SetL2SessionsUp(int32(recvMetric.GetCounters().GetLevel2().GetSessionsUp()))
		isisMetric.SetL2SessionFlap(int32(recvMetric.GetCounters().GetLevel2().GetSessionsFlap()))
		isisMetric.SetL2BroadcastHellosSent(int32(recvMetric.GetCounters().GetLevel2().GetOutBcastHellos()))
		isisMetric.SetL2BroadcastHellosReceived(int32(recvMetric.GetCounters().GetLevel2().GetInBcastHellos()))
		isisMetric.SetL2PointToPointHellosSent(int32(recvMetric.GetCounters().GetLevel2().GetOutP2PHellos()))
		isisMetric.SetL2PointToPointHellosReceived(int32(recvMetric.GetCounters().GetLevel2().GetInP2PHellos()))
		isisMetric.SetL2LspSent(int32(recvMetric.GetCounters().GetLevel2().GetOutLsp()))
		isisMetric.SetL2LspReceived(int32(recvMetric.GetCounters().GetLevel2().GetInLsp()))
		isisMetric.SetL2DatabaseSize(int32(recvMetric.GetCounters().GetLevel2().GetDatabaseSize()))
	}
	return metrics, nil
}

func GetIPv4NeighborStates(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.StatesResponseNeighborsv4StateIter, error) {
	defer Timer(time.Now(), "Getting IPv4 Neighbor states GNMI")
	ethNeighborMap := make(map[string][]string)
	ethernetNames := []string{}
	for _, d := range c.Devices().Items() {
		for _, eth := range d.Ethernets().Items() {
			ethernetNames = append(ethernetNames, eth.Name())
			if _, found := ethNeighborMap[eth.Name()]; !found {
				ethNeighborMap[eth.Name()] = []string{}
			}
			for _, ipv4Address := range eth.Ipv4Addresses().Items() {
				ethNeighborMap[eth.Name()] = append(ethNeighborMap[eth.Name()], ipv4Address.Gateway())
			}
		}
	}

	states := gosnappi.NewApi().NewGetStatesResponse().StatusCode200().Ipv4Neighbors()
	for _, ethernetName := range ethernetNames {
		log.Printf("Fetching IPv4 Neighbor states for ethernet: %v", ethernetName)
		for _, address := range ethNeighborMap[ethernetName] {
			recvState := otg.Telemetry().Interface(ethernetName).Ipv4Neighbor(address).Get(t)
			states.Add().
				SetEthernetName(ethernetName).
				SetIpv4Address(recvState.GetIpv4Address()).
				SetLinkLayerAddress(recvState.GetLinkLayerAddress())
		}
	}
	return states, nil
}

func GetIPv6NeighborStates(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.StatesResponseNeighborsv6StateIter, error) {
	defer Timer(time.Now(), "Getting IPv6 Neighbor states GNMI")
	ethNeighborMap := make(map[string][]string)
	ethernetNames := []string{}
	for _, d := range c.Devices().Items() {
		for _, eth := range d.Ethernets().Items() {
			ethernetNames = append(ethernetNames, eth.Name())
			if _, found := ethNeighborMap[eth.Name()]; !found {
				ethNeighborMap[eth.Name()] = []string{}
			}
			for _, ipv6Address := range eth.Ipv6Addresses().Items() {
				ethNeighborMap[eth.Name()] = append(ethNeighborMap[eth.Name()], ipv6Address.Gateway())
			}
		}
	}

	states := gosnappi.NewApi().NewGetStatesResponse().StatusCode200().Ipv6Neighbors()
	for _, ethernetName := range ethernetNames {
		log.Printf("Fetching IPv6 Neighbor states for ethernet: %v", ethernetName)
		for _, address := range ethNeighborMap[ethernetName] {
			recvState := otg.Telemetry().Interface(ethernetName).Ipv6Neighbor(address).Get(t)
			states.Add().
				SetEthernetName(ethernetName).
				SetIpv6Address(recvState.GetIpv6Address()).
				SetLinkLayerAddress(recvState.GetLinkLayerAddress())
		}
	}
	return states, nil
}

func GetIPv4NeighborMacEntry(t *testing.T, interfaceName string, ipAddress string, otg *ondatra.OTG) (string, error) {
	entries := otg.Telemetry().Interface(interfaceName).Ipv4Neighbor(ipAddress).LinkLayerAddress().Get(t)
	return entries, nil
}

func GetAllIPv4NeighborMacEntries(t *testing.T, otg *ondatra.OTG) ([]string, error) {
	macEntries := otg.Telemetry().InterfaceAny().Ipv4NeighborAny().LinkLayerAddress().Get(t)
	return macEntries, nil
}

func GetAllIPv6NeighborMacEntries(t *testing.T, otg *ondatra.OTG) ([]string, error) {
	macEntries := otg.Telemetry().InterfaceAny().Ipv6NeighborAny().LinkLayerAddress().Get(t)
	return macEntries, nil
}

func GetBGPPrefix(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.StatesResponseBgpPrefixesStateIter, error) {
	states := gosnappi.NewApi().NewGetStatesResponse().StatusCode200().BgpPrefixes()
	for _, d := range c.Devices().Items() {
		bgp := d.Bgp()
		for _, ip := range bgp.Ipv4Interfaces().Items() {
			for _, peer := range ip.Peers().Items() {
				log.Printf("Getting bgp prefix for peer %s\n", peer.Name())

				bgpv4PeerState := states.Add().SetBgpPeerName(peer.Name())
				ipv4Prefixes := otg.Telemetry().BgpPeer(peer.Name()).UnicastIpv4PrefixAny().Get(t)
				stateIpv4Prefix := bgpv4PeerState.Ipv4UnicastPrefixes()
				for _, ipv4Prefix := range ipv4Prefixes {
					prefix := stateIpv4Prefix.Add().SetIpv4Address(ipv4Prefix.GetAddress()).
						SetIpv4NextHop(ipv4Prefix.GetNextHopIpv4Address()).
						SetIpv6NextHop(ipv4Prefix.GetNextHopIpv6Address()).
						SetOrigin(gosnappi.BgpPrefixIpv4UnicastStateOriginEnum(strings.ToLower(ipv4Prefix.GetOrigin().String()))).
						SetPrefixLength(int32(ipv4Prefix.GetPrefixLength()))
					if ipv4Prefix.PathId != nil {
						prefix.SetPathId(int32(ipv4Prefix.GetPathId()))
					}
				}

				ipv6Prefixes := otg.Telemetry().BgpPeer(peer.Name()).UnicastIpv6PrefixAny().Get(t)
				stateIpv6Prefix := bgpv4PeerState.Ipv6UnicastPrefixes()
				for _, ipv6Prefix := range ipv6Prefixes {
					prefix := stateIpv6Prefix.Add().SetIpv6Address(ipv6Prefix.GetAddress()).
						SetIpv4NextHop(ipv6Prefix.GetNextHopIpv4Address()).
						SetIpv6NextHop(ipv6Prefix.GetNextHopIpv6Address()).
						SetOrigin(gosnappi.BgpPrefixIpv6UnicastStateOriginEnum(strings.ToLower(ipv6Prefix.GetOrigin().String()))).
						SetPrefixLength(int32(ipv6Prefix.GetPrefixLength()))
					if ipv6Prefix.PathId != nil {
						prefix.SetPathId(int32(ipv6Prefix.GetPathId()))
					}
				}
			}
		}

		for _, ipv6 := range bgp.Ipv6Interfaces().Items() {
			for _, peer := range ipv6.Peers().Items() {
				log.Printf("Getting bgp prefix for peer %s\n", peer.Name())

				bgpv6PeerState := states.Add().SetBgpPeerName(peer.Name())
				ipv4Prefixes := otg.Telemetry().BgpPeer(peer.Name()).UnicastIpv4PrefixAny().Get(t)
				stateIpv4Prefix := bgpv6PeerState.Ipv4UnicastPrefixes()
				for _, ipv4Prefix := range ipv4Prefixes {
					prefix := stateIpv4Prefix.Add().SetIpv4Address(ipv4Prefix.GetAddress()).
						SetIpv4NextHop(ipv4Prefix.GetNextHopIpv4Address()).
						SetIpv6NextHop(ipv4Prefix.GetNextHopIpv6Address()).
						SetOrigin(gosnappi.BgpPrefixIpv4UnicastStateOriginEnum(strings.ToLower(ipv4Prefix.GetOrigin().String()))).
						SetPrefixLength(int32(ipv4Prefix.GetPrefixLength()))
					if ipv4Prefix.PathId != nil {
						prefix.SetPathId(int32(ipv4Prefix.GetPathId()))
					}
				}

				ipv6Prefixes := otg.Telemetry().BgpPeer(peer.Name()).UnicastIpv6PrefixAny().Get(t)
				stateIpv6Prefix := bgpv6PeerState.Ipv6UnicastPrefixes()
				for _, ipv6Prefix := range ipv6Prefixes {
					prefix := stateIpv6Prefix.Add().SetIpv6Address(ipv6Prefix.GetAddress()).
						SetIpv4NextHop(ipv6Prefix.GetNextHopIpv4Address()).
						SetIpv6NextHop(ipv6Prefix.GetNextHopIpv6Address()).
						SetOrigin(gosnappi.BgpPrefixIpv6UnicastStateOriginEnum(strings.ToLower(ipv6Prefix.GetOrigin().String()))).
						SetPrefixLength(int32(ipv6Prefix.GetPrefixLength()))
					if ipv6Prefix.PathId != nil {
						prefix.SetPathId(int32(ipv6Prefix.GetPathId()))
					}
				}
			}
		}
	}
	return states, nil
}

func GetLagMetric(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponseLagMetricIter, error) {
	defer Timer(time.Now(), "GetLagMetrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().LagMetrics()
	for _, l := range c.Lags().Items() {
		log.Printf("Getting Lag metrics for lag %s\n", l.Name())
		lagMetric := metrics.Add()
		recvMetric := otg.Telemetry().Lag(l.Name()).Get(t)
		lagMetric.SetName(l.Name())
		lagMetric.SetOperStatus(gosnappi.LagMetricOperStatusEnum(strings.ToLower(recvMetric.GetOperStatus().String())))
		lagMetric.SetMemberPortsUp(int32(recvMetric.GetCounters().GetMemberPortsUp()))
		lagMetric.SetBytesRx(int64(recvMetric.GetCounters().GetInOctets()))
		lagMetric.SetBytesTx(int64(recvMetric.GetCounters().GetOutOctets()))
		lagMetric.SetFramesRx(int64(recvMetric.GetCounters().GetInFrames()))
		lagMetric.SetFramesTx(int64(recvMetric.GetCounters().GetOutFrames()))
		lagMetric.SetFramesRxRate(ygot.BinaryToFloat32(recvMetric.GetInRate()))
		lagMetric.SetFramesTxRate(ygot.BinaryToFloat32(recvMetric.GetOutRate()))
	}
	return metrics, nil
}

func GetLacpMetric(t *testing.T, otg *ondatra.OTG, c gosnappi.Config) (gosnappi.MetricsResponseLacpLagMemberMetricIter, error) {
	defer Timer(time.Now(), "GetLacpMetrics GNMI")
	metrics := gosnappi.NewApi().NewGetMetricsResponse().StatusCode200().LacpLagMemberMetrics()
	for _, l := range c.Lags().Items() {
		lagPorts := l.Ports().Items()
		for _, lagPort := range lagPorts {
			log.Printf("Getting Lacp metrics for port %s in lag %s\n", lagPort.PortName(), l.Name())
			lacpMetric := metrics.Add()
			recvMetric := otg.Telemetry().Lacp().LagMember(lagPort.PortName()).Get(t)
			lacpMetric.SetLagName(l.Name())
			lacpMetric.SetLagMemberPortName(lagPort.PortName())
			lacpMetric.SetActivity(gosnappi.LacpLagMemberMetricActivityEnum(strings.ToLower(recvMetric.GetActivity().String())))
			lacpMetric.SetAggregatable(recvMetric.GetAggregatable())
			lacpMetric.SetCollecting(recvMetric.GetCollecting())
			lacpMetric.SetDistributing(recvMetric.GetDistributing())
			lacpMetric.SetLacpInPkts(int64(recvMetric.GetCounters().GetLacpInPkts()))
			lacpMetric.SetLacpOutPkts(int64(recvMetric.GetCounters().GetLacpOutPkts()))
			lacpMetric.SetLacpRxErrors(int64(recvMetric.GetCounters().GetLacpRxErrors()))
			lacpMetric.SetPortNum(int32(recvMetric.GetPortNum()))
			lacpMetric.SetPartnerPortNum(int32(recvMetric.GetPartnerPortNum()))
			lacpMetric.SetTimeout(gosnappi.LacpLagMemberMetricTimeoutEnum(strings.ToLower(recvMetric.GetTimeout().String())))
			lacpMetric.SetPartnerKey(int32(recvMetric.GetPartnerKey()))
			lacpMetric.SetPartnerId(recvMetric.GetPartnerId())
			lacpMetric.SetSystemId(recvMetric.GetSystemId())
			lacpMetric.SetSynchronization(gosnappi.LacpLagMemberMetricSynchronizationEnum(strings.ToLower(recvMetric.GetSynchronization().String())))
		}
	}
	return metrics, nil
}
