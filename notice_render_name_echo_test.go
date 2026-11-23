package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestRenderRefusesAnArgumentWithoutEchoingItsName(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "count kind under a name that is not count",
			input: fmt.Sprintf(`{"code":%d,"args":[{"name":%q,"kind":%d,"value":5}]}`, uint16(CodeSessionNotFound), receivedSecret, uint8(ArgCount)),
		},
		{
			name:  "undeclared name",
			input: fmt.Sprintf(`{"code":%d,"args":[{"name":%q,"kind":%d,"value":"x"}]}`, uint16(CodeSessionNotFound), receivedSecret, uint8(ArgText)),
		},
		{
			name:  "unknown kind",
			input: fmt.Sprintf(`{"code":%d,"args":[{"name":%q,"kind":%d,"value":"x"}]}`, uint16(CodeSessionNotFound), receivedSecret, unknownArgKind),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notice Notice
			if err := json.Unmarshal([]byte(tt.input), &notice); err != nil {
				t.Fatalf("decoding the notice: %v", err)
			}
			_, err := NewRenderer().Render(i18n.English, notice)
			if !errors.Is(err, ErrNoticeArgs) {
				t.Fatalf("error = %v, want ErrNoticeArgs", err)
			}
			echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
		})
	}
}
