// netaddr provides access to the bound network adapters.
package netaddr

import (
	"net"

	"github.com/pkg/errors"
)

func IPv4() ([]net.Addr, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var addrs []net.Addr
	for _, i := range ifs {
		if i.Flags&net.FlagUp == net.FlagUp {
			a, err := i.Addrs()
			if err != nil {
				return nil, errors.WithStack(err)
			}
			for _, a := range a {
				ip := a.(*net.IPNet).IP
				if ip.To4() == nil || ip.IsLoopback() {
					continue
				}
				addrs = append(addrs, a)
			}
		}
	}
	return addrs, nil
}
