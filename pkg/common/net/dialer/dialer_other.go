//go:build !linux

package dialer

func bindDevice(fd uintptr, ifceName string) error {
	return nil
}
