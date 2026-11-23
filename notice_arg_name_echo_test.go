package contracts

import (
	"fmt"
	"testing"
)

func TestArgUnmarshalJSONRefusesAWrongValueWithoutEchoingTheName(t *testing.T) {
	data := fmt.Sprintf(`{"name":%q,"kind":%d,"value":5}`, receivedSecret, uint8(ArgText))
	var arg Arg
	err := arg.UnmarshalJSON([]byte(data))
	if err == nil {
		t.Fatal("a number is accepted as a text value")
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
