package contracts

// SkillNative is implemented by a backend that discovers and loads skills itself
// (e.g. the claude CLI, which reads .claude/skills and ~/.claude/skills). The
// host skips its own skill menu injection and marker detection for such a
// backend so skills are not loaded twice.
type SkillNative interface {
	NativeSkills() bool
}
