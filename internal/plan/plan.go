package plan

type Item struct {
	Text   string
	Detail string
}

type Plan struct {
	Title   string
	Action  string
	Will    []Item
	WillNot []Item
}
