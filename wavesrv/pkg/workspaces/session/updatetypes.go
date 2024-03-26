package session

type ActiveSessionIdUpdate string

func (ActiveSessionIdUpdate) GetType() string {
	return "activesessionid"
}
