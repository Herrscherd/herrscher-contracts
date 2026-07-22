package contracts

import "testing"

type skillNativeStub struct{}

func (skillNativeStub) NativeSkills() bool { return true }

func TestSkillNativeSatisfied(t *testing.T) {
	var _ SkillNative = skillNativeStub{}
}
