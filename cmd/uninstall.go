package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/ca"
	"github.com/scottzirkel/routa/internal/paths"
	"github.com/scottzirkel/routa/internal/systemd"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Reverse `routa install` — stop services, remove units, untrust CA",
	Long: `Stops routa-caddy and routa-dns, removes their unit files, and untrusts
the local CA. By default it keeps routa state and installed PHP builds. Pass
--purge to remove routa-owned XDG state/data/config directories as well. Purge
does not delete your website/project directories referenced by tracked dirs or links.`,
	RunE: runUninstall,
}

var purge bool

func init() {
	uninstallCmd.Flags().BoolVar(&purge, "purge", false, "also delete data/state/config directories")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(_ *cobra.Command, _ []string) error {
	for _, u := range routaUnitsForUninstall() {
		fmt.Printf("→ disable %s\n", u)
		_ = systemd.DisableNow(u) // ignore: unit may not exist
		_ = os.Remove(filepath.Join(paths.SystemdUserDir(), u))
	}
	_ = os.Remove(filepath.Join(paths.SystemdUserDir(), "routa-php@.service"))
	_ = systemd.DaemonReload()

	fmt.Println("→ untrust Caddy local CA (will sudo)")
	if err := ca.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: %v\n", err)
	}

	if purge {
		for _, d := range routaDirsForPurge() {
			fmt.Printf("→ rm -rf %s\n", d)
			if err := purgeRoutaDir(d); err != nil {
				return err
			}
		}
	}
	fmt.Println("Done.")
	return nil
}

func routaUnitsForUninstall() []string {
	units := []string{"routa-caddy.service", "routa-dns.service"}
	return append(units, phpUnitsForUninstall(paths.SystemdUserDir(), paths.RunDir())...)
}

func routaDirsForPurge() []string {
	return []string{paths.DataDir(), paths.StateDir(), paths.ConfigDir()}
}

func purgeRoutaDir(dir string) error {
	if filepath.Base(dir) != "routa" {
		return fmt.Errorf("refusing to purge non-routa directory: %s", dir)
	}
	return os.RemoveAll(dir)
}

func phpUnitsForUninstall(systemdDir, runDir string) []string {
	seen := map[string]bool{}
	addSpec := func(spec string) {
		if spec == "" {
			return
		}
		seen["routa-php@"+spec+".service"] = true
	}

	for _, pattern := range []string{
		filepath.Join(systemdDir, "default.target.wants", "routa-php@*.service"),
		filepath.Join(systemdDir, "routa-php@*.service"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			unit := filepath.Base(match)
			spec := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-php@"), ".service")
			addSpec(spec)
		}
	}

	for _, pattern := range []string{
		filepath.Join(runDir, "php-fpm-*.conf"),
		filepath.Join(runDir, "php-fpm-*.sock"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			base := filepath.Base(match)
			spec := strings.TrimPrefix(base, "php-fpm-")
			spec = strings.TrimSuffix(spec, filepath.Ext(spec))
			addSpec(spec)
		}
	}

	units := make([]string, 0, len(seen))
	for unit := range seen {
		units = append(units, unit)
	}
	sort.Strings(units)
	return units
}
