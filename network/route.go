package main

import (
	"errors"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

// DirectRouting 查看与对应的ip是否是直接连接
func DirectRouting(ip net.IP) (bool, error) {
	routes, err := netlink.RouteGet(ip)
	if err != nil {
		return false, err
	}
	if len(routes) == 1 && routes[0].Gw == nil {
		return true, nil
	}
	// 对应ip和该主机不是直连，中间经过了路由才能实现连接，两者不在同一个网络
	return false, nil
}

// GetDefaultGatewayInterface 获取默认网关
func GetDefaultGatewayInterface() (*net.Interface, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return nil, err
	}

	for _, route := range routes {
		if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
			if route.LinkIndex <= 0 {
				return nil, errors.New("found default route but could not determine interface")
			}
			return net.InterfaceByIndex(route.LinkIndex)
		}
	}

	return nil, errors.New("unable to find default route")
}

func getIfaceAddrs(iface *net.Interface) ([]netlink.Addr, error) {
	link := &netlink.Device{
		LinkAttrs: netlink.LinkAttrs{
			Index: iface.Index,
		},
	}

	return netlink.AddrList(link, syscall.AF_INET)
}

func getIfaceV6Addrs(iface *net.Interface) ([]netlink.Addr, error) {
	link := &netlink.Device{
		LinkAttrs: netlink.LinkAttrs{
			Index: iface.Index,
		},
	}

	return netlink.AddrList(link, syscall.AF_INET6)
}

func GetInterfaceIP4AddrMatch(iface *net.Interface, matchAddr net.IP) error {
	addrs, err := getIfaceAddrs(iface)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		// Attempt to parse the address in CIDR notation
		// and assert it is IPv4
		if addr.IP.To4() != nil {
			if addr.IP.To4().Equal(matchAddr) {
				return nil
			}
		}
	}

	return errors.New("No IPv4 address found for given interface")
}

func GetInterfaceIP6AddrMatch(iface *net.Interface, matchAddr net.IP) error {
	addrs, err := getIfaceV6Addrs(iface)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		// Attempt to parse the address in CIDR notation
		// and assert it is IPv6
		if addr.IP.To16() != nil {
			if addr.IP.To16().Equal(matchAddr) {
				return nil
			}
		}
	}

	return errors.New("No IPv6 address found for given interface")
}
