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
// LANG, as POSIX does: the first of them that is set and not empty decides. It
// reports false when none is set, and when the one that decides names a language
// that is not supported, without looking at the variables after it. getenv is
// injected, so the caller decides where the environment comes from: os.Getenv in a
// client binary.
func FromEnv(getenv func(name string) string) (Language, bool) {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := getenv(name)
		if value == "" {
			continue
		}
		lang, err := ParseLanguage(value)
		if err != nil {
			return 0, false
		}
		return lang, true
	}
	return 0, false
}
