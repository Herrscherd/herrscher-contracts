package contracts

import (
	"context"
	"testing"
)

// stubControl proves a type can satisfy the grown SessionControl interface —
// the compile-time contract every gateway codes against.
type stubControl struct {
	submitted []Inbound
	picked    []string
}

func (s *stubControl) Dispatch(context.Context, []string) (string, error)    { return "", nil }
func (s *stubControl) Create(context.Context, CreateSession) (string, error) { return "", nil }
func (s *stubControl) Close(context.Context, string, bool) (string, error)   { return "", nil }
func (s *stubControl) Sessions() []SessionInfo                               { return nil }
func (s *stubControl) Scrollback(string) []ScrollbackLine                    { return nil }
func (s *stubControl) Resume(string) error                                   { return nil }
func (s *stubControl) Interrupt(string) bool                                 { return false }
func (s *stubControl) Submit(_ string, in Inbound) bool {
	s.submitted = append(s.submitted, in)
	return true
}
func (s *stubControl) Pick(_, value string) bool                { s.picked = append(s.picked, value); return true }
func (s *stubControl) Repos(context.Context) ([]RepoRef, error) { return nil, nil }

var _ SessionControl = (*stubControl)(nil)

func TestSubmitCarriesAttachments(t *testing.T) {
	var ctrl SessionControl = &stubControl{}
	ok := ctrl.Submit("s1", Inbound{
		Conversation: Conversation{Gateway: "discord", ID: "c1"},
		Author:       "leo",
		Text:         "fix this",
		Attachments:  []Attachment{{Filename: "shot.png", URL: "https://cdn/x.png"}},
	})
	if !ok {
		t.Fatal("Submit reported no live session")
	}
	got := ctrl.(*stubControl).submitted
	if len(got) != 1 || len(got[0].Attachments) != 1 || got[0].Attachments[0].Filename != "shot.png" {
		t.Fatalf("attachments did not survive the seam: %+v", got)
	}
}

func TestCreateSessionAdoptsAChannel(t *testing.T) {
	spec := CreateSession{Name: "ch-123", ChannelID: "123"}
	if spec.ChannelID != "123" {
		t.Fatalf("ChannelID = %q, want 123", spec.ChannelID)
	}
}

func TestRepoRefDistinguishesLocalFromRemote(t *testing.T) {
	local := RepoRef{Name: "herrscher", Description: "workspace", Local: true}
	remote := RepoRef{Name: "Herrscherd/dctl", Description: "github"}
	if !local.Local || remote.Local {
		t.Fatalf("Local flag wrong: %+v %+v", local, remote)
	}
}

func TestInboundCarriesAuthorID(t *testing.T) {
	in := Inbound{Author: "leo", AuthorID: "42"}
	if in.AuthorID != "42" {
		t.Fatalf("AuthorID = %q, want 42", in.AuthorID)
	}
}

func TestSessionInfoCarriesIncarnation(t *testing.T) {
	const want = "session-42-v3"
	info := SessionInfo{Incarnation: want}
	if info.Incarnation != want {
		t.Fatalf("Incarnation = %q, want %q", info.Incarnation, want)
	}
}
