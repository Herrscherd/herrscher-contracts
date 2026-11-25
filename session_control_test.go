package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type stubControl struct {
	submitted []Inbound
	picked    []string
}

func (s *stubControl) Dispatch(context.Context, []string) (Result, error) { return Result{}, nil }

func (s *stubControl) Create(context.Context, CreateSession) (Result, error) { return Result{}, nil }

func (s *stubControl) Close(context.Context, string, bool) (Result, error) { return Result{}, nil }
func (s *stubControl) Sessions() []SessionInfo                             { return nil }
func (s *stubControl) Scrollback(string) []ScrollbackLine                  { return nil }
func (s *stubControl) Resume(string) error                                 { return nil }
func (s *stubControl) Interrupt(string) bool                               { return false }
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
	stub, isStub := ctrl.(*stubControl)
	if !isStub {
		t.Fatalf("control is %T, want *stubControl", ctrl)
	}
	got := stub.submitted
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

func TestSessionInfoRoundTripsAPullRequest(t *testing.T) {
	info := SessionInfo{
		Name: "s1",
		PullRequest: SessionPullRequest{
			Number: 7,
			State:  PullRequestMerged,
			Draft:  false,
			URL:    "https://github.com/Herrscherd/herrscher-contracts/pull/7",
		},
	}
	encoded, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"PullRequest":{"Number":7,"State":"merged","Draft":false,"URL":"https://github.com/Herrscherd/herrscher-contracts/pull/7"}`) {
		t.Fatalf("wire JSON = %s, want the pull request in its text form", encoded)
	}
	var back SessionInfo
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
	}
	if back.PullRequest != info.PullRequest {
		t.Fatalf("PullRequest = %+v, want %+v", back.PullRequest, info.PullRequest)
	}
}

func TestSessionInfoRoundTripsWithoutAPullRequest(t *testing.T) {
	info := SessionInfo{Name: "s1"}
	encoded, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "PullRequest") {
		t.Fatalf("wire JSON = %s, want no PullRequest key", encoded)
	}
	var back SessionInfo
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
	}
	if back.PullRequest != (SessionPullRequest{}) {
		t.Fatalf("PullRequest = %+v, want the zero value", back.PullRequest)
	}
}

func TestDecodeRefusesAPullRequestWithoutAState(t *testing.T) {
	var info SessionInfo
	err := json.Unmarshal([]byte(`{"PullRequest":{"Number":7}}`), &info)
	if !errors.Is(err, ErrInvalidSessionPullRequest) || !errors.Is(err, ErrUnknownPullRequestState) {
		t.Fatalf("json.Unmarshal() error = %v, want ErrInvalidSessionPullRequest and ErrUnknownPullRequestState", err)
	}
}

func TestDecodeRefusesANonPositiveNumberWithAState(t *testing.T) {
	for _, payload := range []string{
		`{"PullRequest":{"Number":-3,"State":"open"}}`,
		`{"PullRequest":{"Number":0,"State":"open"}}`,
	} {
		var info SessionInfo
		if err := json.Unmarshal([]byte(payload), &info); !errors.Is(err, ErrInvalidSessionPullRequest) {
			t.Errorf("json.Unmarshal(%s) error = %v, want ErrInvalidSessionPullRequest", payload, err)
		}
	}
}

func TestEncodeRefusesANumberZeroWithFields(t *testing.T) {
	for _, pr := range []SessionPullRequest{
		{State: PullRequestOpen},
		{URL: "https://example.invalid/pull/0"},
	} {
		if _, err := json.Marshal(pr); !errors.Is(err, ErrInvalidSessionPullRequest) {
			t.Errorf("json.Marshal(%+v) error = %v, want ErrInvalidSessionPullRequest", pr, err)
		}
	}
}

func TestEncodeRefusesANegativeNumber(t *testing.T) {
	_, err := json.Marshal(SessionPullRequest{Number: -3, State: PullRequestOpen})
	if !errors.Is(err, ErrInvalidSessionPullRequest) {
		t.Fatalf("json.Marshal() error = %v, want ErrInvalidSessionPullRequest", err)
	}
}

func TestSessionInfoDecodesAnOlderPayloadWithoutPullRequest(t *testing.T) {
	var info SessionInfo
	if err := json.Unmarshal([]byte(`{"Name":"s1","Vendor":"claude"}`), &info); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if info.Name != "s1" || info.PullRequest != (SessionPullRequest{}) {
		t.Fatalf("decoded = %+v, want the name and no pull request", info)
	}
}

func TestSessionInfoRefusesAnUnknownPullRequestState(t *testing.T) {
	var info SessionInfo
	err := json.Unmarshal([]byte(`{"PullRequest":{"Number":7,"State":"draft"}}`), &info)
	if !errors.Is(err, ErrInvalidSessionPullRequest) || !errors.Is(err, ErrUnknownPullRequestState) {
		t.Fatalf("json.Unmarshal() error = %v, want ErrInvalidSessionPullRequest and ErrUnknownPullRequestState", err)
	}
}

func TestSessionPullRequestRefusesToMarshalAStateless(t *testing.T) {
	_, err := json.Marshal(SessionPullRequest{Number: 7, URL: "https://example.invalid/pull/7"})
	if !errors.Is(err, ErrInvalidSessionPullRequest) || !errors.Is(err, ErrUnknownPullRequestState) {
		t.Fatalf("json.Marshal() error = %v, want ErrInvalidSessionPullRequest and ErrUnknownPullRequestState", err)
	}
}

func TestSessionPullRequestZeroValueEncodesAsNullAndDecodesBack(t *testing.T) {
	encoded, err := json.Marshal(SessionPullRequest{})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(encoded) != "null" {
		t.Fatalf("wire JSON = %s, want null", encoded)
	}
	var back SessionPullRequest
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
	}
	if back != (SessionPullRequest{}) {
		t.Fatalf("decoded = %+v, want the zero value", back)
	}
}

func TestSessionPullRequestZeroValueInASliceRoundTrips(t *testing.T) {
	encoded, err := json.Marshal([]SessionPullRequest{{}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var back []SessionPullRequest
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
	}
	if len(back) != 1 || back[0] != (SessionPullRequest{}) {
		t.Fatalf("decoded = %+v from %s, want one zero value", back, encoded)
	}
}

func TestSessionPullRequestDecodeOfAnUnknownStateMatchesBothSentinels(t *testing.T) {
	var pr SessionPullRequest
	err := json.Unmarshal([]byte(`{"Number":3,"State":"draft"}`), &pr)
	if !errors.Is(err, ErrInvalidSessionPullRequest) || !errors.Is(err, ErrUnknownPullRequestState) {
		t.Fatalf("json.Unmarshal() error = %v, want ErrInvalidSessionPullRequest and ErrUnknownPullRequestState", err)
	}
}
