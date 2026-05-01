package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/dev"
	"github.com/scottzirkel/routa/internal/site"
)

var (
	devName    string
	devPort    int
	devTimeout time.Duration
)

var devCmd = &cobra.Command{
	Use:   "dev [name] [-- command...]",
	Short: "Run a detected dev server and proxy <name>.test to it",
	Long: `Runs a local dev server, waits for its HTTP port, and registers a routa
reverse proxy so the app is available at https://<name>.test. WebSockets and
HMR flow through the same proxy.

Without a command, routa detects common project types such as package.json dev
scripts, Rails, Phoenix, and Django. For anything else, pass a command after --.`,
	Args: cobra.ArbitraryArgs,
	RunE: runDev,
}

func init() {
	devCmd.Flags().StringVar(&devName, "name", "", "site name to use when passing a custom command without a positional name")
	devCmd.Flags().IntVar(&devPort, "port", 0, "port to proxy instead of auto-detecting")
	devCmd.Flags().DurationVar(&devTimeout, "timeout", 30*time.Second, "how long to wait for the dev server port")
	rootCmd.AddCommand(devCmd)
}

func runDev(_ *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	name := filepath.Base(cwd)
	if devName != "" {
		name = devName
	}
	var command []string
	var kind string
	defaultPort := 0

	switch {
	case len(args) == 0:
		spec, err := dev.Detect(cwd)
		if err != nil {
			return err
		}
		kind, command, defaultPort = spec.Kind, spec.Command, spec.DefaultPort
	case devName != "":
		command = args
		kind = "custom"
	case len(args) == 1:
		name = args[0]
		spec, err := dev.Detect(cwd)
		if err != nil {
			return err
		}
		kind, command, defaultPort = spec.Kind, spec.Command, spec.DefaultPort
	default:
		name = args[0]
		command = args[1:]
		kind = "custom"
	}

	name, err = normalizeSiteName(name)
	if err != nil {
		return err
	}
	if len(command) == 0 {
		return fmt.Errorf("empty dev command")
	}
	if devPort != 0 {
		defaultPort = devPort
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	ports := make(chan int, 8)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", strings.Join(command, " "), err)
	}
	exited := make(chan struct{})
	var procErr error
	go func() {
		procErr = cmd.Wait()
		close(exited)
	}()
	go streamDevOutput(stdout, os.Stdout, ports)
	go streamDevOutput(stderr, os.Stderr, ports)

	port, err := waitForDevPort(ctx, defaultPort, ports, exited, &procErr, devTimeout)
	if err != nil {
		_ = terminateProcess(cmd.Process)
		<-exited
		return err
	}
	target := fmt.Sprintf("127.0.0.1:%d", port)
	if err := registerDevProxy(name, target); err != nil {
		_ = terminateProcess(cmd.Process)
		<-exited
		return err
	}

	label := kind
	if label == "" {
		label = "dev"
	}
	fmt.Printf("dev %s: https://%s.test → %s (%s)\n", label, name, target, strings.Join(command, " "))
	<-exited
	err = procErr
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func registerDevProxy(name, target string) error {
	s, err := site.Load()
	if err != nil {
		return err
	}
	site.AddLink(s, site.Link{Name: name, Target: target, Secure: true})
	return commitAndReload(s, fmt.Sprintf("proxy %s.test → %s", name, target))
}

var portRE = regexp.MustCompile(`(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\]|:)\D*([1-9][0-9]{1,4})`)

func streamDevOutput(src io.Reader, dst *os.File, ports chan<- int) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(dst, line)
		if m := portRE.FindStringSubmatch(line); len(m) == 2 {
			if p, err := strconv.Atoi(m[1]); err == nil && p <= 65535 {
				select {
				case ports <- p:
				default:
				}
			}
		}
	}
}

func waitForDevPort(ctx context.Context, defaultPort int, ports <-chan int, exited <-chan struct{}, procErr *error, timeout time.Duration) (int, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(150 * time.Millisecond)
	defer tick.Stop()

	candidates := []int{}
	if defaultPort != 0 {
		candidates = append(candidates, defaultPort)
	}
	seen := map[int]bool{}
	for _, p := range candidates {
		seen[p] = true
	}

	for {
		for _, port := range candidates {
			if canDialLocal(port) {
				return port, nil
			}
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-exited:
			if *procErr != nil {
				return 0, fmt.Errorf("dev command exited before opening a port: %w", *procErr)
			}
			return 0, fmt.Errorf("dev command exited before opening a port")
		case <-deadline.C:
			if defaultPort == 0 {
				return 0, fmt.Errorf("timed out waiting for a dev server port; pass --port when the command does not print one")
			}
			return 0, fmt.Errorf("timed out waiting for dev server on 127.0.0.1:%d", defaultPort)
		case port := <-ports:
			if !seen[port] {
				candidates = append(candidates, port)
				seen[port] = true
			}
		case <-tick.C:
		}
	}
}

func canDialLocal(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func terminateProcess(p *os.Process) error {
	if p == nil {
		return nil
	}
	if err := p.Signal(os.Interrupt); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	return p.Kill()
}
