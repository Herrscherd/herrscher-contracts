// Package i18n is the runtime of herrscher's translated texts: the closed set of
// supported languages, their parsing from a locale, the resolution of a language
// from ordered sources, and a catalog of go-i18n messages that falls back to
// English. It is imported only by the modules that render text, and the root
// contracts package imports it too, for the shared catalog of notices, so
// go-i18n and x/text reach every module that imports contracts.
package i18n
