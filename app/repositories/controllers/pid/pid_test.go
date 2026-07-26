package pid_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/pid"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// t0 is the zero time; adding durations to it gives predictable dt values.
var t0 = time.Time{}

func cfg(kp, ki, kd, min, max, iClamp, setpoint float64) config.Config {
	return config.Config{PID: config.PID{Kp: kp, Ki: ki, Kd: kd, Min: min, Max: max, IClamp: iClamp, Setpoint: setpoint}}
}

func TestProportionalOnly(t *testing.T) {
	// Kp=1, Ki=0, Kd=0 → output = Kp * error = 1 * (15-5) = 10
	c := pid.New(cfg(1.0, 0, 0, 0, 100, 20, 15.0))
	out, err := c.Compute(5.0, t0.Add(5*time.Second)) // dt=5s, measured=5, setpoint=15
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0, got %f", out)
	}
}

func TestOutputClamped(t *testing.T) {
	// Error=50, Kp=2 → raw output=100, but max=50
	c := pid.New(cfg(2.0, 0, 0, 0.5, 50, 20, 50.0))
	out, err := c.Compute(0.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 50.0 {
		t.Errorf("expected 50.0 (clamped), got %f", out)
	}
}

func TestMinimumClamp(t *testing.T) {
	// CPU above setpoint → negative output → clamped to min=0.5
	c := pid.New(cfg(1.0, 0, 0, 0.5, 50, 20, 15.0))
	out, err := c.Compute(30.0, t0.Add(5*time.Second)) // error = 15-30 = -15
	if err != nil {
		t.Fatal(err)
	}
	if out != 0.5 {
		t.Errorf("expected 0.5 (min clamp), got %f", out)
	}
}

func TestIntegralAntiWindup(t *testing.T) {
	// Ki=1, dt=5, error=5 for many ticks → I accumulates but is clamped to IClamp=20
	c := pid.New(cfg(0, 1.0, 0, 0, 100, 20, 15.0))
	for i := 0; i < 20; i++ {
		// error=5, dt=5 each tick; unclamped I would be 500
		c.Compute(10.0, t0.Add(time.Duration(i+1)*5*time.Second))
	}
	// I should be clamped at 20, so output = I = 20
	out, err := c.Compute(15.0, t0.Add(21*5*time.Second)) // error=0, P=0, D=0 → output=I
	if err != nil {
		t.Fatal(err)
	}
	if out > 25 || out < 15 {
		t.Errorf("integral windup: expected ~20, got %f", out)
	}
}

func TestDerivative(t *testing.T) {
	// Kd=1, dt=5: error goes from 10 to 5 → D = Kd*(5-10)/5 = -1
	c := pid.New(cfg(0, 0, 1.0, -100, 100, 20, 15.0))
	c.Compute(5.0, t0.Add(5*time.Second))               // first tick: error=10, D=0 (no prev)
	out, err := c.Compute(10.0, t0.Add(10*time.Second)) // second tick: error=5, D=(5-10)/5*Kd=-1
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-(-1.0)) > 0.001 {
		t.Errorf("expected D=-1.0, got %f", out)
	}
}

func TestFeedforward(t *testing.T) {
	// Kp=Ki=Kd=0, feedforward only: u_ff = setpoint/Kff = 50/2.5 = 20.
	c := pid.New(config.Config{PID: config.PID{Max: 100, IClamp: 20, Setpoint: 50, FfGain: 2.5}})
	out, err := c.Compute(0, t0.Add(1*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-20.0) > 0.001 {
		t.Errorf("expected feedforward 20.0, got %f", out)
	}
}

func TestFeedforwardDisabledWhenZero(t *testing.T) {
	// FfGain=0 -> no feedforward; pure proportional: out = Kp*e = 1*(50-0) clamp.
	c := pid.New(cfg(1.0, 0, 0, 0, 100, 20, 50.0))
	out, err := c.Compute(0, t0.Add(1*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-50.0) > 0.001 {
		t.Errorf("expected 50.0 (no feedforward), got %f", out)
	}
}

func TestReset(t *testing.T) {
	c := pid.New(cfg(1.0, 1.0, 1.0, 0, 100, 20, 15.0))
	c.Compute(5.0, t0.Add(5*time.Second))
	c.Reset()
	out1, err := c.Compute(5.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	c2 := pid.New(cfg(1.0, 1.0, 1.0, 0, 100, 20, 15.0))
	out2, err := c2.Compute(5.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out1-out2) > 0.001 {
		t.Errorf("reset: expected same output as fresh controller, got %f vs %f", out1, out2)
	}
}

func TestSetSetpoint(t *testing.T) {
	c := pid.New(cfg(1.0, 0, 0, 0, 100, 20, 10.0))
	if err := c.SetSetpoint(20.0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := c.Compute(10.0, t0.Add(5*time.Second)) // error = 20-10 = 10, P = 10
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0 after setpoint change, got %f", out)
	}
}

func TestSetSetpointNaN(t *testing.T) {
	c := pid.New(cfg(1.0, 0, 0, 0, 100, 20, 10.0))
	if err := c.SetSetpoint(math.NaN()); err == nil {
		t.Error("expected error for NaN setpoint, got nil")
	}
}

func TestComputeNaNMeasured(t *testing.T) {
	c := pid.New(cfg(1.0, 0, 0, 0, 100, 20, 15.0))
	_, err := c.Compute(math.NaN(), t0.Add(5*time.Second))
	if err == nil {
		t.Error("expected error for NaN measured, got nil")
	}
}

func TestComputeOutdatedTime(t *testing.T) {
	c := pid.New(cfg(1.0, 0, 0, 0, 100, 20, 15.0))
	c.Compute(5.0, t0.Add(10*time.Second))
	_, err := c.Compute(5.0, t0.Add(5*time.Second)) // earlier than last compute
	if err == nil {
		t.Error("expected error for outdated measureTime, got nil")
	}
}

func TestNewPanicsMinGtMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for min > max")
		}
	}()
	pid.New(cfg(1.0, 0, 0, 100, 50, 20, 15.0))
}

func TestNewPanicsNegativeIClamp(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative iClamp")
		}
	}()
	pid.New(cfg(1.0, 0, 0, 0, 100, -1, 15.0))
}

func TestSetpointFilterMovingAverageRampaReferencia(t *testing.T) {
	cfg := config.Config{PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100,
		Setpoint: 50, SetpointFilterType: "moving_average", SetpointFilterSize: 4}}
	c := pid.New(cfg)
	base := time.Now()
	out1, _ := c.Compute(0, base.Add(1*time.Second))
	if math.Abs(out1-12.5) > 1e-6 {
		t.Fatalf("1º tique: referência deve ser 12,5 (P=12,5), got %v", out1)
	}
}

func TestLegacyTauViraExponencialEquivalente(t *testing.T) {
	cfg := config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 5000},
		PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100, Setpoint: 50,
			SetpointFilterTau: 10}}
	c := pid.New(cfg)
	base := time.Now()
	out1, _ := c.Compute(0, base.Add(5*time.Second))
	want := (1 - math.Exp(-0.5)) * 50
	if math.Abs(out1-want) > 1e-6 {
		t.Fatalf("tau=2T deve equivaler a size=2: got %v want %v", out1, want)
	}
}

func TestSetParamsTrocaFiltroParaMovingAverageSemErro(t *testing.T) {
	cfg := config.Config{PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100,
		Setpoint: 100, SetpointFilterType: "moving_average", SetpointFilterSize: 4}}
	c := pid.New(cfg)
	base := time.Now()
	if _, err := c.Compute(0, base.Add(1*time.Second)); err != nil {
		t.Fatal(err)
	}

	kind, size := "moving_average", 8
	if err := c.SetParams(entities.ControllerParams{SetpointFilterType: &kind, SetpointFilterSize: &size}); err != nil {
		t.Fatalf("esperava trocar tipo/tamanho do filtro sem erro, got %v", err)
	}
}

func TestSetParamsTrocaFiltroRampaDaReferenciaEfetivaAtual(t *testing.T) {
	cfg := config.Config{PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100,
		Setpoint: 100, SetpointFilterType: "moving_average", SetpointFilterSize: 4}}
	c := pid.New(cfg)
	base := time.Now()
	out0, err := c.Compute(0, base.Add(1*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out0-25) > 1e-6 {
		t.Fatalf("referência efetiva esperada 25 antes da troca de filtro, got %v", out0)
	}

	kind, size := "exponential", 4
	if err := c.SetParams(entities.ControllerParams{SetpointFilterType: &kind, SetpointFilterSize: &size}); err != nil {
		t.Fatal(err)
	}

	out1, err := c.Compute(0, base.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	a := math.Exp(-1.0 / 4.0)
	want := a*25 + (1-a)*100
	if math.Abs(out1-want) > 1e-6 {
		t.Fatalf("após a troca, a referência deve continuar de 25 (Seed), got %v want %v", out1, want)
	}
	if math.Abs(out1) < 1e-6 {
		t.Fatalf("referência não deveria saltar para 0 após a troca de filtro, got %v", out1)
	}
}

func TestSetParamsFiltroTipoInvalidoRetornaErroEMantemFiltroAntigo(t *testing.T) {
	cfg := config.Config{PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100,
		Setpoint: 100, SetpointFilterType: "moving_average", SetpointFilterSize: 4}}
	c := pid.New(cfg)
	base := time.Now()
	if _, err := c.Compute(0, base.Add(1*time.Second)); err != nil {
		t.Fatal(err)
	}

	bogus := "bogus"
	if err := c.SetParams(entities.ControllerParams{SetpointFilterType: &bogus}); err == nil {
		t.Fatal("esperava erro para tipo de filtro inválido")
	}

	out, err := c.Compute(0, base.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-50) > 1e-6 {
		t.Fatalf("filtro antigo (moving_average) deveria continuar funcionando, got %v", out)
	}
}
