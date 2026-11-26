package contracts

import "testing"

func TestModelReadableFollowsTheTypeThenTheExtension(t *testing.T) {
	cases := []struct {
		name string
		att  Attachment
		want bool
	}{
		{"image", Attachment{ContentType: "image/png"}, true},
		{"text with a charset", Attachment{ContentType: " Text/Plain; charset=utf-8"}, true},
		{"pdf", Attachment{ContentType: "application/pdf", Filename: "x.bin"}, true},
		{"yaml", Attachment{ContentType: "application/yaml"}, true},
		{"zip", Attachment{ContentType: "application/zip", Filename: "notes.txt"}, false},
		{"word", Attachment{ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}, false},
		{"extension without a type", Attachment{Filename: "Plan.MD"}, true},
		{"unknown extension without a type", Attachment{Filename: "bundle.tar.gz"}, false},
		{"nothing", Attachment{}, false},
		{"blank type with a readable name", Attachment{ContentType: " ", Filename: "a.png"}, false},
		{"parameters only with a readable name", Attachment{ContentType: ";charset=utf-8", Filename: "a.txt"}, false},
		{"padded type wins over the name", Attachment{ContentType: " application/zip ", Filename: "a.png"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ModelReadable(tc.att); got != tc.want {
				t.Fatalf("ModelReadable(%+v) = %v, want %v", tc.att, got, tc.want)
			}
		})
	}
}
