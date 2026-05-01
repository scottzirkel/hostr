package diag

import "testing"

func TestRequiredBinaryAbsenceFails(t *testing.T) {
	check := checkRequiredBinary("definitely-not-installed-routa-test-bin", "install it")

	if check.Status != Fail {
		t.Fatalf("status = %q, want %q", check.Status, Fail)
	}
	if check.Hint != "install it" {
		t.Fatalf("hint = %q", check.Hint)
	}
}

func TestOptionalBinaryAbsenceIsNonBlocking(t *testing.T) {
	check := checkOptionalBinary("definitely-not-installed-routa-test-bin", "optional")

	if check.Status != Absent {
		t.Fatalf("status = %q, want %q", check.Status, Absent)
	}
}
