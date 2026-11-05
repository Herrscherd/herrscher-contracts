package i18n

// Resolve returns the first candidate that is a supported language, and English
// when none is. A zero candidate is an undefined source, so a host passes every
// source it has, most specific first.
func Resolve(candidates ...Language) Language {
	for _, candidate := range candidates {
		if candidate.supported() {
			return candidate
		}
	}
	return English
}

// FromEnv reads the language of the client's locale from LC_ALL, LC_MESSAGES and
// LANG, in that order, and returns the first of them that names a supported
// language. It reports false when none does. getenv is injected, so the caller
// decides where the environment comes from: os.Getenv in a client binary.
func FromEnv(getenv func(name string) string) (Language, bool) {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if lang, err := ParseLanguage(getenv(name)); err == nil {
			return lang, true
		}
	}
	return 0, false
}
