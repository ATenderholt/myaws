package docker

type EnsureImageProgressDetail struct {
	Current int
	Total   int
}

type EnsureImageProgress struct {
	Status         string
	ProgressDetail EnsureImageProgressDetail
	Progress       string
	ID             string
}

func (p EnsureImageProgress) String() string {
	if len(p.ID) > 0 {
		return p.ID + " " + p.Status + " " + p.Progress
	} else {
		return p.Status
	}
}
