package factors

type shiftedSnapshot struct {
	inner Snapshot
	shift map[string]float64
}

var _ Snapshot = &shiftedSnapshot{}

func (s *shiftedSnapshot) Get(key string) (float64, bool, error) {
	v, ok, err := s.inner.Get(key)
	shift := s.shift[key]
	v += shift
	return v, ok, err
}

func Shift(snapshot Snapshot, shift map[string]float64) Snapshot {
	return &shiftedSnapshot{
		inner: snapshot,
		shift: shift,
	}
}
