package contracts

type skillNativeStub struct{}

func (skillNativeStub) NativeSkills() bool { return true }

var _ SkillNative = skillNativeStub{}
