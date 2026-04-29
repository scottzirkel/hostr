package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDoctorReportJSONShape(t *testing.T) {
	report := doctorReport{
		Services: []doctorService{
			{Name: "hostr-dns.service", OK: true, Status: "active"},
		},
		Network: doctorNetwork{
			CaddyAdmin: doctorEndpoint{Name: "caddy admin", OK: true, Detail: "127.0.0.1:2019 (up)"},
			CaddyHTTPS: doctorEndpoint{Name: "caddy https", OK: true, Detail: "127.0.0.1:443 (Phase 2)"},
			HostrDNS:   doctorEndpoint{Name: "hostr-dns", OK: true, Detail: "127.0.0.1:1053 (up)"},
		},
		DNS: doctorDNS{
			OK:       true,
			Name:     "doctor.hostr.test",
			Answer:   "127.0.0.1",
			Expected: "127.0.0.1",
		},
		Cutover: doctorCutover{
			Phase: "phase_two",
			Label: "Phase 2",
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{
		`"services"`,
		`"network"`,
		`"dns"`,
		`"cutover"`,
		`"phase":"phase_two"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("JSON missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, "site_probes") {
		t.Fatalf("empty site probes should be omitted: %s", body)
	}
}

func TestDoctorProbeJSONShape(t *testing.T) {
	report := doctorReport{
		SiteProbes: []doctorProbeResult{
			{Name: "app", URL: "https://app.test", OK: false, Status: "error", Error: "connection refused"},
			{Name: "api", URL: "https://api.test", OK: true, Status: "HTTP 204", StatusCode: 204},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{
		`"site_probes"`,
		`"error":"connection refused"`,
		`"status_code":204`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("JSON missing %s: %s", want, body)
		}
	}
}
