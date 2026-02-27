package pnwforecast

import "fmt"

type SkillError struct {
	Kind string
	Msg  string
}

func (e *SkillError) Error() string {
	return fmt.Sprintf("%s: %s", e.Kind, e.Msg)
}

func (e *SkillError) Payload() ErrorPayload {
	return ErrorPayload{Type: e.Kind, Message: e.Msg}
}

func NewSkillError(kind, msg string) *SkillError {
	return &SkillError{Kind: kind, Msg: msg}
}
