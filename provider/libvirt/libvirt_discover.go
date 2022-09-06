package libvirt_discover

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"

	"libvirt.org/go/libvirt"
)

type Provider struct {
}

func (p *Provider) Help() string {
	return `Libvirt:

    provider:          "libvirt"
    qemu_uri:          The QEMU URI to connect to
    ns_identifier:     XML Namespace Identifier
    metadata_key:      The metadata key to filter on
    metadata_value:    The metadata value to filter on
`
}

func (p *Provider) Addrs(args map[string]string, l *log.Logger) ([]string, error) {
	if args["provider"] != "libvirt" {
		return nil, fmt.Errorf("discover-libvirt:invalid provider  " + args["provider"])
	}

	if l == nil {
		l = log.New(ioutil.Discard, "", 0)
	}

	qemu_uri := args["qemu_uri"]
	nsIdentifier := args["ns_identifier"]
	metadataKey := args["metadata_key"]
	metadataValue := args["metadata_value"]

	conn, err := libvirt.NewConnect(qemu_uri)
	if err != nil {
		return nil, fmt.Errorf("Cant connect to libvirt server")
	}
	defer conn.Close()

	doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		return nil, err
	}

	type Metadata struct {
		Key   string `xml:"key"`
		Value string `xml:"value"`
	}

	var metadata Metadata
	var addrs []string

	for _, dom := range doms {
		l.Printf("[INFO] discover-libvirt: Filter domains with %s=%s", metadataKey, metadataValue)
		domName, err := dom.GetName()
		domMetadata, err := dom.GetMetadata(2, nsIdentifier, 0)
		if err != nil {
			return nil, err
		}

		xml.Unmarshal([]byte(domMetadata), &metadata)

		if metadata.Key != metadataKey {
			return nil, fmt.Errorf("Metadata key not found")
		}

		if metadata.Value != metadataValue {
			return nil, fmt.Errorf("Metadata value not found under given key")
		}

		interfaces, err := dom.ListAllInterfaceAddresses(1)
		if err != nil {
			return nil, fmt.Errorf("No network interfaces on this domain: %s", domName)
		}

		if len(interfaces) > 1 {
			if len(interfaces[1].Addrs) > 1 {
				l.Printf("[DEBUG] discover-libvirt: Domain %s has IPv4 address: %v", domName, interfaces[1].Addrs[0].Addr)
				addrs = append(addrs, interfaces[1].Addrs[0].Addr)
			}
		}
		dom.Free()
	}
	l.Printf("[DEBUG] discover-libvirt: Found ip addresses: %v", addrs)
	return addrs, nil
}
