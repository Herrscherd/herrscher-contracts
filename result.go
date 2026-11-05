package contracts

// Result is what a command hands back: a Notice for the human who asked, or typed
// data for a caller that renders it itself, such as the --json flag or a gateway
// that lists sessions. The zero value carries neither.
type Result struct {
	notice *Notice
	data   any
}

// NoticeResult returns a result that tells the human notice.
func NoticeResult(notice Notice) Result {
	return Result{notice: &notice}
}

// DataResult returns a result that carries data for the caller to render.
func DataResult(data any) Result {
	return Result{data: data}
}

// Notice returns the notice of the result, and false when it carries data or
// nothing.
func (r Result) Notice() (Notice, bool) {
	if r.notice == nil {
		return Notice{}, false
	}
	return *r.notice, true
}

// Data returns the data of the result, or nil when it carries a notice or nothing.
func (r Result) Data() any {
	return r.data
}
