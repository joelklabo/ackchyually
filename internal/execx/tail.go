package execx

type Tail struct {
	buf []byte
	cap int
}

func NewTail(maxBytes int) *Tail {
	return &Tail{cap: maxBytes}
}

func (t *Tail) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.cap {
		t.buf = t.buf[len(t.buf)-t.cap:]
	}
	return len(p), nil
}

func (t *Tail) String() string { return string(t.buf) }
