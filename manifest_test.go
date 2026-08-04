package contracts

import "testing"

func TestManifestCarriesCapabilities(t *testing.T) {
	m := Manifest{
		Kind:         "discord",
		Category:     CategoryGateway,
		Capabilities: Capabilities{Reactions: true, SelectMenus: true, Replies: true},
	}
	if m.Category != "gateway" {
		t.Fatalf("CategoryGateway should be %q, got %q", "gateway", m.Category)
	}
	if !m.Capabilities.SelectMenus {
		t.Fatalf("capabilities not carried by manifest")
	}
}

// A gateway that declares no attachment hosts must read as "allow nothing", not
// as "unset, so allow anything": the field is the host's SSRF allowlist, and the
// zero value is the one every existing manifest already has.
func TestManifestAttachmentHostsDefaultToNone(t *testing.T) {
	var m Manifest
	if len(m.AttachmentHosts) != 0 {
		t.Fatalf("AttachmentHosts = %v, want empty by default", m.AttachmentHosts)
	}
	m.AttachmentHosts = []string{"cdn.discordapp.com", "media.discordapp.net"}
	if len(m.AttachmentHosts) != 2 || m.AttachmentHosts[0] != "cdn.discordapp.com" {
		t.Fatalf("AttachmentHosts = %v, not carried by manifest", m.AttachmentHosts)
	}
}
