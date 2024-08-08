package worker

type Task struct {
	ID int
}

type Result struct {
	Response string
	Error    error
}

type queueItem struct {
	task Task
	ch   chan Result
}
