// Package i18n is the runtime of herrscher's translated texts: the closed set of
// supported languages, their parsing from a locale, the resolution of a language
// from ordered sources, and a catalog of go-i18n messages that falls back to
// English. It is imported only by the modules that render text, so the root
// contracts package keeps the standard library as its only dependency.
package i18n
