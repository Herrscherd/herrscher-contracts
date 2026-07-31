package contracts

import "testing"

// The zero value must read as "live": every manifest written before Status
// existed omits the field, and those plugins were all shipping.
func TestStatusZeroValueIsLive(t *testing.T) {
	var m Manifest
	if m.Status != StatusLive {
		t.Fatalf("zero Status = %q, want StatusLive", m.Status)
	}
	if got := m.Status.String(); got != "live" {
		t.Fatalf("zero Status.String() = %q, want %q", got, "live")
	}
}

func TestStatusString(t *testing.T) {
	for _, tc := range []struct {
		status Status
		want   string
	}{
		{StatusLive, "live"},
		{StatusWIP, "wip"},
		{StatusExperimental, "experimental"},
		{StatusDeprecated, "deprecated"},
	} {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("Status(%q).String() = %q, want %q", string(tc.status), got, tc.want)
		}
	}
}

// Status must not disturb registration or category lookup — it is descriptive
// only, and a wip plugin is still discovered like any other.
func TestStatusDoesNotAffectDiscovery(t *testing.T) {
	r := &Registry{}
	r.Register(Plugin{Manifest: Manifest{Kind: "a", Category: CategoryBackend}})
	r.Register(Plugin{Manifest: Manifest{Kind: "b", Category: CategoryBackend, Status: StatusWIP}})
	if got := len(r.Backends()); got != 2 {
		t.Fatalf("Backends() = %d, want 2", got)
	}
}
