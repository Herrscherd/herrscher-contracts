package contracts

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

type noticeRecorder struct {
	recGateway

	notified []Notice
	langs    []i18n.Language
}

func (n *noticeRecorder) Notify(_ context.Context, _ Conversation, lang i18n.Language, notice Notice) (MessageID, error) {
	n.langs = append(n.langs, lang)
	n.notified = append(n.notified, notice)
	return "", nil
}

func degradedNoticeSink(t *testing.T, gateway Gateway) NoticeSink {
	t.Helper()
	sink, ok := Degrade(gateway).(NoticeSink)
	if !ok {
		t.Fatal("degraded gateway must satisfy NoticeSink")
	}
	return sink
}

func TestDegradeNoticeRendersTextForGatewayWithoutSink(t *testing.T) {
	rec := &recGateway{manifest: Manifest{Capabilities: full()}}
	sink := degradedNoticeSink(t, rec)
	notice := Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "x")}}
	if _, err := sink.Notify(context.Background(), Conversation{}, i18n.French, notice); err != nil {
		t.Fatal(err)
	}
	if len(rec.posts) != 1 || rec.posts[0] != "aucune session nommée « x »" {
		t.Fatalf("posts = %q, want the French text only", rec.posts)
	}
}

func TestDegradeNoticeForwardsToSinkWithLanguage(t *testing.T) {
	rec := &noticeRecorder{recGateway: recGateway{manifest: Manifest{Kind: "sink"}}}
	sink := degradedNoticeSink(t, rec)
	notice := Notice{Code: CodeUnknownAgent, Args: []Arg{Text("Name", "x")}}
	if _, err := sink.Notify(context.Background(), Conversation{}, i18n.French, notice); err != nil {
		t.Fatal(err)
	}
	if len(rec.notified) != 1 || rec.notified[0].Code != CodeUnknownAgent || rec.langs[0] != i18n.French {
		t.Fatalf("sink received %+v in %v", rec.notified, rec.langs)
	}
	if len(rec.posts) != 0 {
		t.Fatalf("a gateway with a sink must not receive a text fallback: %q", rec.posts)
	}
}

func TestDegradeNoticeRefusesMalformedArgWithoutPosting(t *testing.T) {
	rec := &recGateway{manifest: Manifest{Capabilities: full()}}
	sink := degradedNoticeSink(t, rec)
	_, err := sink.Notify(context.Background(), Conversation{}, i18n.English, Notice{Code: CodeSessionNotFound, Args: []Arg{{}}})
	if !errors.Is(err, ErrNoticeArgs) || len(rec.posts) != 0 {
		t.Fatalf("Notify err = %v, posts = %q; want ErrNoticeArgs and no post", err, rec.posts)
	}
}

type fallbackSink struct {
	recGateway
}

func (f *fallbackSink) Notify(_ context.Context, _ Conversation, _ i18n.Language, _ Notice) (MessageID, error) {
	return "", fmt.Errorf("discord notify: %w: %w", ErrNoticeFallbackPosted, errors.New("render failed"))
}

func TestDegradeNoticeNeverPostsOverAFallback(t *testing.T) {
	rec := &fallbackSink{recGateway: recGateway{manifest: Manifest{Kind: "sink"}}}
	sink := degradedNoticeSink(t, rec)
	_, err := sink.Notify(context.Background(), Conversation{}, i18n.French, Notice{Code: CodeCompacted})
	if !errors.Is(err, ErrNoticeFallbackPosted) {
		t.Fatalf("Notify err = %v, want ErrNoticeFallbackPosted", err)
	}
	if len(rec.posts) != 0 {
		t.Fatalf("Degrade must not post over a fallback the sink already posted: %q", rec.posts)
	}
}
