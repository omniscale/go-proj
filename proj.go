/*
Package proj transforms coordinates with Proj.

	// New Proj by EPSG code.
	wgs84, err := proj.NewEPSG(4326)
	if err != nil {
		log.Fatal(err)
	}

	// Proj by definition string.
	utm32, err := proj.New("epsg:25832")
	if err != nil {
		log.Fatal(err)
	}

	pts := []proj.Coord{
		proj.XY(53.2, 8.15),
		proj.XY(52.32, 9.12),
	}

	// Transform all coordinates to UTM 32 (in-place).
	if err := wgs84.Transform(utm32, pts); err != nil {
		log.Fatal(err)
	}


	// All coordinates are expected to be in EPSG axis order.
	// Call NormalizeForVisualization if your coordinates are always in lon/lat, E/N order.
	wgs84.NormalizeForVisualization()
	if err := wgs84.Transform(utm32, []proj.Coord{proj.XY(8.15, 53.2)}); err != nil {
		log.Fatal(err)
	}


	// Transformer from src to dst projection.
	transf, err := proj.NewTransformer("epsg:25832", "epsg:3857")
	if err != nil {
		log.Fatal(err)
	}
	if err := transf.Transform(pts); err != nil {
		log.Fatal(err)
	}
*/
package proj

// #cgo LDFLAGS: -lproj
// #include <proj.h>
// #include <stdlib.h>
import "C"

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"unsafe"
)

// Proj represents a single coordinate reference system.
type Proj struct {
	p          *C.PJ
	ctx        *C.PJ_CONTEXT
	normalized bool
}

// NewEPSG initializes a new projection by the numeric EPSG code.
func NewEPSG(epsgCode int) (*Proj, error) {
	return New(fmt.Sprintf("epsg:%d", epsgCode))
}

// New initializes new projection with a proj init string (e.g. "epsg:4326", or "+proj=longlat +datum=WGS84 +no_defs").
func New(init string) (*Proj, error) {
	ctx := C.proj_context_create()
	C.proj_log_level(ctx, C.PJ_LOG_NONE)

	c := C.CString(init)
	defer C.free(unsafe.Pointer(c))
	proj := C.proj_create(ctx, c)
	if proj == nil {
		errno := C.proj_context_errno(ctx)
		return nil, errors.New(C.GoString(C.proj_context_errno_string(ctx, errno)))
	}

	p := &Proj{p: proj, ctx: ctx}
	runtime.SetFinalizer(p, free)
	return p, nil
}

func free(p *Proj) {
	p.Free()
}

// Free deallocates the projection immediately. Proj will be deallocated on garbage collection otherwise.
func (p *Proj) Free() {
	if p.p != nil {
		C.proj_destroy(p.p)
		p.p = nil
	}
	if p.ctx != nil {
		C.proj_context_destroy(p.ctx)
		p.ctx = nil
	}
}

// NormalizeForVisualization converts axis order so that coordinates are always
// x/y or long/lat axis order. The EPSG axis order is ignored when calling
// Transform.
func (p *Proj) NormalizeForVisualization() error {
	if p.normalized {
		return nil
	}
	// Try to normalize for visualization.
	normProj := C.proj_normalize_for_visualization(p.ctx, p.p)
	if normProj == nil {
		errno := C.proj_context_errno(p.ctx)
		return errors.New(C.GoString(C.proj_context_errno_string(p.ctx, errno)))
	}

	C.proj_destroy(p.p)
	p.p = normProj
	p.normalized = true
	return nil
}

type Coord struct {
	X, Y float64
	Z    float64
	T    float64
}

func XY(x, y float64) Coord {
	return Coord{X: x, Y: y, Z: 0, T: math.MaxFloat64}
}

// Transform coordinates to dst projection. Transforms coordinates in-place.
func (p *Proj) Transform(dst *Proj, pts []Coord) error {
	if p == nil {
		return errors.New("missing/invalid projection")
	}
	if dst == nil {
		return errors.New("missing/invalid dst projection")
	}
	if pts == nil {
		return nil
	}

	tr := C.proj_create_crs_to_crs_from_pj(p.ctx, p.p, dst.p, nil, nil)

	r := C.proj_trans_array(tr, C.PJ_FWD, C.ulong(len(pts)), (*C.PJ_COORD)(unsafe.Pointer(&pts[0])))

	if r != 0 {
		errnoRef := C.proj_context_errno(p.ctx)
		if errnoRef == 0 {
			return errors.New("unknown error")
		}
		return errors.New(C.GoString(C.proj_context_errno_string(p.ctx, errnoRef)))
	}

	return nil
}

// IsLatLong returns whether the projection uses lat/long coordinates, instead projected.
func (p *Proj) IsLatLong() bool {
	tp := C.proj_get_type(p.p)
	return tp == C.PJ_TYPE_GEODETIC_CRS || tp == C.PJ_TYPE_GEOGRAPHIC_2D_CRS || tp == C.PJ_TYPE_GEOGRAPHIC_3D_CRS
}

// Definition returns projection description.
func (p *Proj) Description() string {
	info := C.proj_pj_info(p.p)
	return strings.TrimSpace(C.GoString(info.description))
}

func (p *Proj) String() string {
	return "Proj(" + p.Description() + ")"
}

// Unit returns the unit name of the first axis.
// Can return degree, meter or foot, but also long names like 'US survey foot'. Returns empty string if there is no unit name, or if there was an error.
func (p *Proj) UnitName() string {
	var unitName *C.char = nil

	crs := C.proj_crs_get_coordinate_system(p.ctx, p.p)
	defer C.proj_destroy(crs)

	r := C.proj_cs_get_axis_info(p.ctx, crs, 0,
		nil,       // out_name
		nil,       // out_abbrev
		nil,       // out_direction
		nil,       // out_unit_conv_factor
		&unitName, // out_unit_name
		nil,       // out_unit_auth_name
		nil,       // out_unit_code
	)

	if r == 0 {
		return ""
	}
	if unitName != nil {
		return C.GoString(unitName)
	}
	return ""
}

// Transformer projects coordinates from Src to Dst.
type Transformer struct {
	Src *Proj
	Dst *Proj
}

// Transform coordinates fron src to dst projection. Transforms coordinates in-place.
func (t *Transformer) Transform(pts []Coord) error {
	return t.Src.Transform(t.Dst, pts)
}

func (t *Transformer) NormalizeForVisualization() error {
	if err := t.Src.NormalizeForVisualization(); err != nil {
		return err
	}
	return t.Dst.NormalizeForVisualization()
}

// NewTransformer initializes new transformer with src and dst projection with
// a full proj4 init string (e.g. "+proj=longlat +datum=WGS84 +no_defs").
func NewTransformer(initSrc, initDst string) (Transformer, error) {
	src, err := New(initSrc)
	if err != nil {
		return Transformer{}, err
	}
	dst, err := New(initDst)
	if err != nil {
		return Transformer{}, err
	}
	return Transformer{Src: src, Dst: dst}, nil
}

// NewEPSGTransformer initializes a new transformer with src and dst projection by the numeric EPSG code.
func NewEPSGTransformer(srcEPSG, dstEPSG int) (Transformer, error) {
	src, err := NewEPSG(srcEPSG)
	if err != nil {
		return Transformer{}, err
	}
	dst, err := NewEPSG(dstEPSG)
	if err != nil {
		return Transformer{}, err
	}
	return Transformer{Src: src, Dst: dst}, nil
}
