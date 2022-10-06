package proj

import (
	"math"
	"os"
	"strings"
	"testing"
)

// Test transformation of a single point with different axis orders.
// 4326 and 31467 use lat/lon N/E axis order, 25832 uses E/N.
// Check that normalization changes the order of 4326 and 31467, but not for 25832.
// PROJ_USE_PROJ4_INIT_RULES should also change order and normalization should have no effect in this case.

func TestTransformUTM(t *testing.T) {
	// lat/lon and E/N
	checkTransform(t, XY(53.2, 8.15), XY(443220.719, 5894856.508), "epsg:4326", "epsg:25832", false)
}
func TestTransformUTM_Normalize(t *testing.T) {
	// lon/lat and E/N
	checkTransform(t, XY(8.15, 53.2), XY(443220.719, 5894856.508), "epsg:4326", "epsg:25832", true)
}

func TestTransformUTM_Proj4Init(t *testing.T) {
	// lat/lon and E/N
	checkProj4InitTransform(t, XY(8.15, 53.2), XY(443220.719, 5894856.508), "+init=epsg:4326", "+init=epsg:25832", false)
}
func TestTransformUTM_Proj4InitNormalize(t *testing.T) {
	// lon/lat and E/N
	checkProj4InitTransform(t, XY(8.15, 53.2), XY(443220.719, 5894856.508), "+init=epsg:4326", "+init=epsg:25832", true)
}
func TestTransformGK(t *testing.T) {
	// lat/lon and N/E
	checkTransform(t, XY(53.2, 8.15), XY(5896773.991, 3443269.238), "epsg:4326", "epsg:31467", false)
}
func TestTransformGK_Normalize(t *testing.T) {
	// lon/lat and E/N
	checkTransform(t, XY(8.15, 53.2), XY(3443269.238, 5896773.991), "epsg:4326", "epsg:31467", true)
}
func TestTransformGK_Proj4Init(t *testing.T) {
	// lat/lon and N/E
	checkProj4InitTransform(t, XY(53.2, 8.15), XY(5896773.991, 3443269.238), "epsg:4326", "epsg:31467", false)
}
func TestTransformGK_Proj4InitNormalize(t *testing.T) {
	// lon/lat and E/N
	checkProj4InitTransform(t, XY(8.15, 53.2), XY(3443269.238, 5896773.991), "epsg:4326", "epsg:31467", true)
}

func checkProj4InitTransform(t *testing.T, src, expected Coord, projA, projB string, normalize bool) {
	os.Setenv("PROJ_USE_PROJ4_INIT_RULES", "YES")
	defer os.Setenv("PROJ_USE_PROJ4_INIT_RULES", "NO")
	checkTransform(t, src, expected, projA, projB, normalize)
}

func checkTransform(t *testing.T, src, expected Coord, projA, projB string, normalize bool) {
	p1, err := New(projA)
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Free()
	p2, err := New(projB)
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Free()

	if normalize {
		if err := p1.NormalizeForVisualization(); err != nil {
			t.Fatal(err)
		}
		if err := p2.NormalizeForVisualization(); err != nil {
			t.Fatal(err)
		}
	}

	pts := []Coord{src}

	if err := p1.Transform(p2, pts); err != nil {
		t.Fatal(err)
	}

	if math.Abs(pts[0].X-expected.X) > 0.01 {
		t.Error(pts)
	}
	if math.Abs(pts[0].Y-expected.Y) > 0.01 {
		t.Error(pts)
	}

	if err := p2.Transform(p1, pts); err != nil {
		t.Fatal(err)
	}

	if math.Abs(pts[0].X-src.X) > 0.0001 {
		t.Error(pts)
	}
	if math.Abs(pts[0].Y-src.Y) > 0.0001 {
		t.Error(pts)
	}
}

func TestTransformError(t *testing.T) {
	p1, err := New("epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	p1.NormalizeForVisualization()
	p2, err := New("epsg:25832")
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Free()
	p2.NormalizeForVisualization()
	pinvalid, err := New("epsg:999999")
	if err == nil {
		t.Fatal("no error for unknown projection")
	}

	pts := []Coord{
		XY(8.15, 53.2),
	}

	if err := p1.Transform(pinvalid, pts); err == nil {
		t.Error("no err from transformation with nil")
	}

	if err := pinvalid.Transform(p1, pts); err == nil {
		t.Error("no err from transformation with nil")
	}

	if err := p1.Transform(p2, nil); err != nil {
		t.Error("err from transformation with no coordinates")
	}

	pts = []Coord{
		XY(-81.15, 90.1),
	}
	if err := p1.Transform(p2, pts); err == nil || !strings.Contains(err.Error(), "Invalid coordinate") {
		t.Error("no/unexpected err from transformation:", err)
	}
}

func TestNewTransformer(t *testing.T) {
	pts := []Coord{
		XY(8.15, 9.12),
		XY(53.2, 52.32),
	}

	transf, err := NewTransformer("+init=epsg:4326", "+init=epsg:3857")
	if err == nil || !strings.Contains(err.Error(), "Invalid PROJ string") {
		t.Fatal(err)
	}

	os.Setenv("PROJ_USE_PROJ4_INIT_RULES", "YES")
	defer os.Setenv("PROJ_USE_PROJ4_INIT_RULES", "NO")
	transf, err = NewTransformer("+init=epsg:4326", "+init=epsg:3857")
	if err != nil {
		t.Fatal(err)
	}
	if err := transf.Transform(pts); err != nil {
		t.Fatal(err)
	}

	transf, err = NewEPSGTransformer(3857, 4326)
	if err != nil {
		t.Fatal(err)
	}
	if err := transf.Transform(pts); err != nil {
		t.Fatal(err)
	}
}

func TestLatLong(t *testing.T) {
	p, err := New("epsg:4326")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsLatLong() {
		t.Error("epsg:4326 is not LatLong")
	}

	p, err = New("epsg:25832")
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
	os.Setenv("PROJ_USE_PROJ4_INIT_RULES", "YES")
	defer os.Setenv("PROJ_USE_PROJ4_INIT_RULES", "NO")
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

func TestDescription(t *testing.T) {
	var tests = []struct {
		epsg        int
		description string
	}{
		{4326, "WGS 84"},
		{31467, "DHDN / 3-degree Gauss-Kruger zone 3"},
		{2222, "NAD83 / Arizona East (ft)"},
		{2228, "NAD83 / California zone 4 (ftUS)"},
		{2136, "Accra / Ghana National Grid"},
	}
	for _, tt := range tests {
		p, err := NewEPSG(tt.epsg)
		if err != nil {
			t.Error(err)
			return
		}
		if d := p.Description(); d != tt.description {
			t.Error(d)
		}
	}
}

func TestUnitName(t *testing.T) {
	var tests = []struct {
		epsg int
		unit string
	}{
		{4326, "degree"},
		{31467, "metre"},
		{2222, "foot"},
		{2228, "US survey foot"},
		{2136, "Gold Coast foot"},
	}
	for _, tt := range tests {
		p, err := NewEPSG(tt.epsg)
		if err != nil {
			t.Error(err)
			continue
		}
		if u := p.UnitName(); u != tt.unit {
			t.Errorf("%s != %s for %q", u, tt.unit, p)
		}
	}
}

func BenchmarkProj(b *testing.B) {
	pts := []Coord{
		XY(53.1, 8.15),
		XY(53.2, 8.25),
		XY(53.3, 8.75),
		XY(53.3, 8.00),
	}

	p1, err := New("epsg:4326")
	if err != nil {
		b.Fatal(err)
	}
	p2, err := New("epsg:25832")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if err := p1.Transform(p2, pts); err != nil {
			b.Fatal(err)
		}
		if err := p2.Transform(p1, pts); err != nil {
			b.Fatal(err)
		}
	}
	p1.Free()
	p2.Free()
}
