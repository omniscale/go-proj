package proj

import (
	"math"
	"strings"
	"testing"
)

func TestTransform(t *testing.T) {
	p1, err := New("+init=epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Free()
	p2, err := New("+init=epsg:25832")
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Free()

	xs := []float64{8.15}
	ys := []float64{53.2}

	if err := p1.Transform(p2, xs, ys); err != nil {
		t.Fatal(err)
	}

	if math.Abs(xs[0]-443220.719) > 0.01 {
		t.Error(xs)
	}
	if math.Abs(ys[0]-5894856.508) > 0.01 {
		t.Error(ys)
	}

	if err := p2.Transform(p1, xs, ys); err != nil {
		t.Fatal(err)
	}

	if math.Abs(xs[0]-8.15) > 0.0001 {
		t.Error(xs)
	}
	if math.Abs(ys[0]-53.2) > 0.0001 {
		t.Error(ys)
	}
}

func TestTransformError(t *testing.T) {
	p1, err := New("+init=epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := New("+init=epsg:25832")
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Free()
	pinvalid, err := New("+init=epsg:999999")
	if err == nil {
		t.Fatal("no error for unknown projection")
	}

	xs := []float64{8.15}
	ys := []float64{53.2}

	if err := p1.Transform(pinvalid, xs, ys); err == nil {
		t.Error("no err from transformation with nil")
	}

	if err := pinvalid.Transform(p1, xs, ys); err == nil {
		t.Error("no err from transformation with nil")
	}

	xs = []float64{8.15, 9.20}
	if err := p1.Transform(p2, xs, ys); err == nil {
		t.Error("no err from transformation with nil")
	}
	xs = []float64{8.15}
	ys = []float64{53.2, 52.0}
	if err := p1.Transform(p2, xs, ys); err == nil {
		t.Error("no err from transformation with nil")
	}

	if err := p1.Transform(p2, nil, nil); err != nil {
		t.Error("err from transformation with no coordinates")
	}

	xs = []float64{-81.15}
	ys = []float64{90}
	if err := p1.Transform(p2, xs, ys); !strings.Contains(err.Error(), "latitude or longitude exceeded limits") {
		t.Error("no/unexpected err from transformation:", err)
	}
}

func TestLatLong(t *testing.T) {
	p, err := New("+init=epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsLatLong() {
		t.Error("epsg:4326 is not LatLong")
	}

	p, err = New("+init=epsg:25832")
	if err != nil {
		t.Fatal(err)
	}
	if p.IsLatLong() {
		t.Error("epsg:25832 is LatLong")
	}
}

func TestNewEPSG(t *testing.T) {
	p, err := NewEPSG(4326)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("projection is nil")
	}

	p, err = NewEPSG(999999)
	if err == nil {
		t.Fatal("no error for unknown projection")
	}
}

func TestNew(t *testing.T) {
	p, err := New("+init=epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("projection is nil")
	}

	p, err = New("+init=epsg:999999")
	if err == nil {
		t.Fatal("no error for unknown projection")
	}

	p, err = New("")
	if err == nil {
		t.Fatal("no error for empty projection")
	}

	p, err = New(" +proj=utm +zone=32 +ellps=GRS80 +towgs84=0,0,0,0,0,0,0 +units=m +no_defs ")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("projection is nil")
	}

	p, err = New(" +proj=utm +zone=99 +ellps=GRS80 +towgs84=0,0,0,0,0,0,0 +units=m +no_defs ")
	if err == nil {
		t.Fatal("no error for invalid projection")
	}
}

func BenchmarkProj(b *testing.B) {
	xs := []float64{8.15, 8.25, 8.75, 8.00}
	ys := []float64{53.1, 53.2, 53.3, 53.3}

	p1, err := New("+init=epsg:4326")
	if err != nil {
		b.Fatal(err)
	}
	p2, err := New("+init=epsg:25832")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if err := p1.Transform(p2, xs, ys); err != nil {
			b.Fatal(err)
		}
		if err := p2.Transform(p1, xs, ys); err != nil {
			b.Fatal(err)
		}
	}
	p1.Free()
	p2.Free()
}

func TestSetSearchPath(t *testing.T) {
	p1, err := New("+init=epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Free()

	p2, err := New("+init=test-epsg:99999")
	if err == nil {
		t.Fatal("expected error")
	}
	/// test-epsg contains 99999 projection with definition of 25832
	SetSearchPaths([]string{"."})
	p2, err = New("+init=test-epsg:99999")
	if err != nil {
		t.Fatal(err)
	}

	xs := []float64{8.15}
	ys := []float64{53.2}

	if err := p1.Transform(p2, xs, ys); err != nil {
		t.Fatal(err)
	}

	if math.Abs(xs[0]-443220.719) > 0.01 {
		t.Error(xs)
	}
	if math.Abs(ys[0]-5894856.508) > 0.01 {
		t.Error(ys)
	}
}
