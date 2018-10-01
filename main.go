package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func main() {
	var portRaw uint
	var anycastIPRaw string
	flag.UintVar(&portRaw, "port", 0, "Port to look for service listening")
	flag.StringVar(&anycastIPRaw, "anycast-ip", "", "Anycast IP the service wants to listen on")
	flag.Parse()

	if portRaw >= 1<<16 || portRaw < 1 {
		fmt.Printf("Port (%d) out of range [1..%v)", portRaw, 1<<16)
		os.Exit(1)
	}

	port := uint16(portRaw)

	anycastIP := net.ParseIP(anycastIPRaw)
	if !anycastIP.IsGlobalUnicast() {
		fmt.Printf("IP is required and needs to be valid and part of the global unicast range")
		os.Exit(1)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	log.Printf("Anycast Helper Sync loop started")

	netlinkAddr := &netlink.Addr{IPNet: &net.IPNet{IP: anycastIP, Mask: net.CIDRMask(32, 32)}, Label: "anycast0", Scope: unix.RT_SCOPE_UNIVERSE, Flags: unix.IFA_F_PERMANENT}
	for range ticker.C {
		hasListeners := HasListenersOnPortSimple(port)
		link, err := netlink.LinkByName("anycast0")
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			netlink.LinkAdd(&netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "anycast0"}})
		}
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			log.Printf("Failed to list addresses: %v", err)
			continue
		}
		hasIP := false
		for _, addr := range addrs {
			if addr.IP.Equal(anycastIP) {
				hasIP = true
			}
		}
		if hasIP && !hasListeners {
			log.Printf("Lost listener, removing address")
			if err := netlink.AddrDel(link, netlinkAddr); err != nil {
				log.Printf("Failed to delete address: %v", err)
				continue
			}
		}
		if !hasIP && hasListeners {
			log.Printf("Detected listener, adding address")
			if err := netlink.AddrAdd(link, netlinkAddr); err != nil {
				log.Printf("Failed to delete address: %v", err)
				continue
			}
		}
	}
}
