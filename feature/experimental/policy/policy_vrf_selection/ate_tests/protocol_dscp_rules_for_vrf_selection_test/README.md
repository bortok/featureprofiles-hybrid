# RT-3.2: Multiple <Protocol, DSCP> Rules for VRF Selection

## Summary

Ensure that multiple protocol and dscp based VRF selection rules are matched correctly.

## Procedure

Configure DUT with 1 input interface connected to ATE port-1, and a second interface (output) connected to ATE port-2 with VLAN-based subinterfaces, with the following assignments: 

* network-instance “10” corresponding to VLAN 10, default route via VLAN 10 subinterface. 
* network-instance “20” corresponding to VLAN 20, default route via VLAN 20 subinterface. 
* network-instance “30” corresponding to VLAN 30, default route via VLAN 30 subinterface. 

Configure DUT with the following rules and determine measurement: 

### Case #1: 
* Rules: 

    * Protocol IPinIP, DSCP 10 to network-instance 10 
    * Protocol IPinIP, DSCP 20 to network-instance 20 
    * Protocol IPinIP, DSCP 30 to network-instance 30 

Ensure packets with only expected DSCPs reach each egress port. 

### Case #2: 

* Rules: 

    * Protocol IPinIP, DSCP 10, 11, 12 to network-instance 10 
    * Protocol IPinIP, DSCP 20, 21, 22 to network-instance 20 
    * Protocol IPinIP, DSCP 30, 31, 32 to network-instance 30 

Ensure packets with only expected DSCPs reach each egress port. 

### Case #3: 

* Rules: 

    * Protocol IPinIP, DSCP 10, 11, 12 to network-instance 10 
    * Protocol IPinIP, DSCP 10, 11, 12 to network-instance 20 

Ensure that packets are only received in network-instance 10. Ensure that with overlapping rules, first rule takes precedence. 

### Case #4: 

* Rules: 
    * Protocol IPinIP to network-instance 10 
    * Protocol IPinIP, DSCP 20 to network-instance 20 

Ensure that unspecified fields are wildcard and IPinIP packets are only received at VLAN 10 subinterface. 

## Config Parameter Coverage
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/config/type
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/policy-id
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/rules/rule/sequence-id
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/rules/rule/ipv4/config/dscp-set
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/rules/rule/ipv4/config/protocol
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/rules/rule/action/config/network-instance
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/interfaces/interface/interface-id
 *  /openconfig-network-instance/network-instances/network-instance/policy-forwarding/interfaces/interface/config/apply-vrf-selection-policy

## Paths

* /openconfig-network-instance/network-instances/network-instance/policy-forwarding
* /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy
* /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/rules/rule
* /openconfig-network-instance/network-instances/network-instance/policy-forwarding/policies/policy/rules/rule/ipv4
* /openconfig-network-instance/network-instances/network-instance/policy-forwarding/interfaces/interface/config/apply-vrf-selection-policy 

