package control

import "math"

type Model struct {
	K     float64
	Tau   float64
	Theta float64
}

func (m Model) validate(sampleSeconds float64) error {
	if !(m.Tau > 0) || !(sampleSeconds > 0) || !(m.Theta >= 0) {
		return ErrInvalidModel
	}
	if !isFinite(m.K) || !isFinite(m.Tau) || !isFinite(m.Theta) || !isFinite(sampleSeconds) {
		return ErrInvalidModel
	}
	return nil
}

func (m Model) DelaySamples(sampleSeconds float64) int {
	return max(int(math.Round(m.Theta/sampleSeconds)), 0)
}

type predictor struct {
	model  Model
	output float64
	held   float64
	delay  []float64
	head   int
}

func newPredictor(m Model, sampleSeconds float64) *predictor {
	return &predictor{model: m, delay: make([]float64, m.DelaySamples(sampleSeconds))}
}

func (p *predictor) advance(dt float64) (now, delayed float64) {
	a := math.Exp(-dt / p.model.Tau)
	p.output = a*p.output + p.model.K*(1-a)*p.held
	return p.output, p.push(p.output)
}

func (p *predictor) push(y float64) float64 {
	if len(p.delay) == 0 {
		return y
	}
	delayed := p.delay[p.head]
	p.delay[p.head] = y
	p.head = (p.head + 1) % len(p.delay)
	return delayed
}

func (p *predictor) hold(u float64) {
	p.held = u
}
