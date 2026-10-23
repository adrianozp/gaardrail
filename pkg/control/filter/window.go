package filter

type window struct {
	buf []float64
	idx int
	n   int
	sum float64
}

func newWindow(size int) *window {
	return &window{buf: make([]float64, max(size, 1))}
}

func (w *window) push(x float64) float64 {
	size := len(w.buf)
	if size <= 1 {
		return x
	}
	if w.n == size {
		w.sum -= w.buf[w.idx]
	} else {
		w.n++
	}
	w.buf[w.idx] = x
	w.sum += x
	w.idx = (w.idx + 1) % size
	return w.sum / float64(size)
}

func (w *window) fill(v float64) {
	w.n = len(w.buf)
	w.sum = v * float64(len(w.buf))
	w.idx = 0
	for i := range w.buf {
		w.buf[i] = v
	}
}

func (w *window) reset() {
	w.idx = 0
	w.n = 0
	w.sum = 0
	clear(w.buf)
}
