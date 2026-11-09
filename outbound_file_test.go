package contracts

import (
	"context"
	"errors"
	"slices"
	"testing"
)

const testOutbox = "/w/out"

func TestValidateOutboundFileAcceptsAFileInsideTheOutbox(t *testing.T) {
	cases := map[string]OutboundFile{
		"top level":       {Path: "/w/out/pixel.png", Name: "pixel.png", ContentType: "image/png"},
		"nested":          {Path: "/w/out/sheet/frame-1.png", Name: "frame-1.png"},
		"type with param": {Path: "/w/out/notes.txt", Name: "notes.txt", ContentType: "text/plain; charset=utf-8"},
	}
	for name, file := range cases {
		if err := ValidateOutboundFile(file, testOutbox); err != nil {
			t.Fatalf("%s: ValidateOutboundFile(%+v) = %v, want nil", name, file, err)
		}
	}
}

func TestValidateOutboundFileRefusesEachMalformedFile(t *testing.T) {
	cases := map[string]struct {
		file   OutboundFile
		outbox string
	}{
		"relative path":           {OutboundFile{Path: "out/pixel.png", Name: "pixel.png"}, testOutbox},
		"empty path":              {OutboundFile{Name: "pixel.png"}, testOutbox},
		"outside the outbox":      {OutboundFile{Path: "/home/shan/.ssh/id_rsa", Name: "id_rsa"}, testOutbox},
		"parent segment":          {OutboundFile{Path: "/w/out/../../etc/passwd", Name: "passwd"}, testOutbox},
		"sibling with a prefix":   {OutboundFile{Path: "/w/outside/pixel.png", Name: "pixel.png"}, testOutbox},
		"the outbox itself":       {OutboundFile{Path: "/w/out", Name: "out"}, testOutbox},
		"relative outbox":         {OutboundFile{Path: "/w/out/pixel.png", Name: "pixel.png"}, "w/out"},
		"empty outbox":            {OutboundFile{Path: "/w/out/pixel.png", Name: "pixel.png"}, ""},
		"no name":                 {OutboundFile{Path: "/w/out/pixel.png"}, testOutbox},
		"name is the current dir": {OutboundFile{Path: "/w/out/pixel.png", Name: "."}, testOutbox},
		"name is the parent dir":  {OutboundFile{Path: "/w/out/pixel.png", Name: ".."}, testOutbox},
		"slash in name":           {OutboundFile{Path: "/w/out/pixel.png", Name: "out/pixel.png"}, testOutbox},
		"backslash in name":       {OutboundFile{Path: "/w/out/pixel.png", Name: `out\pixel.png`}, testOutbox},
		"bad content type":        {OutboundFile{Path: "/w/out/pixel.png", Name: "pixel.png", ContentType: "image/"}, testOutbox},
	}
	for name, tc := range cases {
		if err := ValidateOutboundFile(tc.file, tc.outbox); !errors.Is(err, ErrInvalidOutboundFile) {
			t.Fatalf("%s: ValidateOutboundFile(%+v, %q) error = %v, want ErrInvalidOutboundFile", name, tc.file, tc.outbox, err)
		}
	}
}

type fileGateway struct {
	recGateway

	sent []OutboundFile
}

func (g *fileGateway) SendFiles(_ context.Context, _ Conversation, _ MessageID, _ string, files []OutboundFile) (MessageID, error) {
	g.sent = append(g.sent, files...)
	return "files", nil
}

func degradeFiles() []OutboundFile {
	return []OutboundFile{{Path: "/w/out/a.png", Name: "a.png"}, {Path: "/w/out/b.png", Name: "b.png"}}
}

func TestDegradeSendFilesForwardsToAFileSender(t *testing.T) {
	gateway := &fileGateway{recGateway: recGateway{manifest: Manifest{Capabilities: full()}}}
	sender, ok := Degrade(gateway).(FileSender)
	if !ok {
		t.Fatal("Degrade(gateway) does not implement FileSender")
	}
	if _, err := sender.SendFiles(context.Background(), Conversation{Gateway: "g", ID: "c"}, "m", "voilà", degradeFiles()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(gateway.sent, degradeFiles()) || len(gateway.replies) != 0 {
		t.Fatalf("sent %+v and replied %v, want the files and no reply", gateway.sent, gateway.replies)
	}
}

func TestDegradeSendFilesListsTheFilesForAPlainGateway(t *testing.T) {
	const listed = "voilà\na.png: /w/out/a.png\nb.png: /w/out/b.png"
	cases := map[string]struct {
		capabilities Capabilities
		wantReplies  []string
		wantPosts    []string
	}{
		"a gateway with replies gets a reply":   {full(), []string{listed}, nil},
		"a gateway without replies gets a post": {Capabilities{}, nil, []string{listed}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			gateway := &recGateway{manifest: Manifest{Capabilities: tc.capabilities}}
			sender, ok := Degrade(gateway).(FileSender)
			if !ok {
				t.Fatal("Degrade(gateway) does not implement FileSender")
			}
			if _, err := sender.SendFiles(context.Background(), Conversation{Gateway: "g", ID: "c"}, "m", "voilà", degradeFiles()); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(gateway.replies, tc.wantReplies) || !slices.Equal(gateway.posts, tc.wantPosts) {
				t.Fatalf("replies %q posts %q, want replies %q posts %q", gateway.replies, gateway.posts, tc.wantReplies, tc.wantPosts)
			}
		})
	}
}
