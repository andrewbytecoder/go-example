package generate

//go:generate stringer -type=Status

type Status int

const (
	Pending Status = iota
	Running
	Failed
)
