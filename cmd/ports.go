package cmd

import "fmt"

func localhostAddr(port string) string {
	return "127.0.0.1:" + port
}

func portInUseError(addr, service string) error {
	return fmt.Errorf("%s is already in use; stop the existing %s process or choose another port", addr, service)
}
