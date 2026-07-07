package plan

type Item struct {
	Key    string
	Data   map[string]any
	Detail string
}

type Plan struct {
	TitleKey string
	Action   string
	Will     []Item
	WillNot  []Item
}
