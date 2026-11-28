package contracts

const maxForgeLoginBytes = 39

// ForgeLogin is the login of an account on a forge, the name a mention carries.
// Its field is unexported, so a value is built only by [ParseForgeLogin]; the
// zero value names no account, which is the filter a listing leaves off.
type ForgeLogin struct {
	text string
}

// ParseForgeLogin checks text against the rules of a forge login: 1 to 39 bytes
// of ASCII letters, digits, hyphens, underscores and dots, not starting with a
// hyphen or a dot. It returns an error matching [ErrInvalidForgeLogin]
// otherwise; the error does not repeat the text.
func ParseForgeLogin(text string) (ForgeLogin, error) {
	if !validForgeLogin(text) {
		return ForgeLogin{}, ErrInvalidForgeLogin
	}
	return ForgeLogin{text: text}, nil
}

// String is the login, empty for the zero value.
func (l ForgeLogin) String() string { return l.text }

// IsZero reports whether l names no account.
func (l ForgeLogin) IsZero() bool { return l.text == "" }

func validForgeLogin(text string) bool {
	if text == "" || len(text) > maxForgeLoginBytes || text[0] == '-' || text[0] == '.' {
		return false
	}
	return allBytes(text, isForgeLoginByte)
}

func isForgeLoginByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-' || b == '_' || b == '.'
}
