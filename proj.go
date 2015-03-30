/*
Package proj transforms coordinates with libproj4.

	// proj by EPSG code
	wgs84, err := proj.NewEPSG(4326)
	if err != nil {
		log.Fatal(err)
	}

	// proj by proj4 definition string
	utm32, err := proj.New("+proj=utm +zone=32 +ellps=GRS80 +towgs84=0,0,0,0,0,0,0 +units=m +no_defs")
	if err != nil {
		log.Fatal(err)
	}

	xs := []float64{8.15, 9.12}
	ys := []float64{53.2, 52.32}

	// transform all coordinates to UTM 32 (in-place)
	if err := wgs84.Transform(utm32, xs, ys); err != nil {
		log.Fatal(err)
	}


	// transformer from src to dst projection
	transf, err := proj.NewTransformer("+init=epsg:25832", "+init=epsg:3857")
	if err != nil {
		log.Fatal(err)
	}
	if err := transf.Transform(xs, ys); err != nil {
		log.Fatal(err)
	}
*/
package proj

// #cgo LDFLAGS: -lproj
// #include <proj_api.h>
// #include <stdlib.h>
// extern char *go_proj_finder_wrapper(char *name);
import "C"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

type Proj struct {
	p   C.projPJ
	ctx C.projCtx
}

const deg2Rad = math.Pi / 180.0
const rad2Deg = 180.0 / math.Pi

// NewEPSG initializes a new projection by the numeric EPSG code.
func NewEPSG(epsgCode int) (*Proj, error) {
	return New(fmt.Sprintf("+init=epsg:%d", epsgCode))
}

// New initializes new projection with a full proj4 init string (e.g. "+proj=longlat +datum=WGS84 +no_defs").
func New(init string) (*Proj, error) {
	ctx := C.pj_ctx_alloc()
	if ctx == nil {
		errnoRef := C.pj_get_errno_ref()
		if errnoRef == nil {
			return nil, errors.New("unknown error on pj_ctx_alloc")
		}
		return nil, errors.New(C.GoString(C.pj_strerrno(*errnoRef)))
	}

	c := C.CString(init)
	defer C.free(unsafe.Pointer(c))
	proj := C.pj_init_plus_ctx(ctx, c)
	if proj == nil {
		errno := C.pj_ctx_get_errno(ctx)
		return nil, errors.New(C.GoString(C.pj_strerrno(errno)))
	}

	p := &Proj{proj, ctx}
	runtime.SetFinalizer(p, free)
	return p, nil
}

func free(p *Proj) {
	p.Free()
}

// Free deallocates the projection immediately. Proj will be deallocated on garbage collection otherwise.
func (p *Proj) Free() {
	if p.p != nil {
		C.pj_free(p.p)
		p.p = nil
	}
	if p.ctx != nil {
		C.pj_ctx_free(p.ctx)
		p.ctx = nil
	}
}

// Transform coordinates to dst projection. Transforms coordinates in-place.
func (p *Proj) Transform(dst *Proj, xs, ys []float64) error {
	if p == nil {
		return errors.New("missing/invalid projection")
	}
	if dst == nil {
		return errors.New("missing/invalid dst projection")
	}
	if len(xs) != len(ys) {
		return errors.New("number of x and y coordinates differs")
	}
	if xs == nil || ys == nil {
		return nil
	}

	if C.pj_is_latlong(p.p) != 0 {
		for i := range xs {
			xs[i] *= deg2Rad
		}
		for i := range ys {
			ys[i] *= deg2Rad
		}
	}
	r := C.pj_transform(p.p, dst.p, C.long(len(xs)), 0,
		(*C.double)(unsafe.Pointer(&xs[0])),
		(*C.double)(unsafe.Pointer(&ys[0])),
		nil)

	if r != 0 {
		errnoRef := C.pj_get_errno_ref()
		if errnoRef == nil {
			return errors.New("unknown error")
		}
		return errors.New(C.GoString(C.pj_strerrno(*errnoRef)))
	}

	if C.pj_is_latlong(dst.p) != 0 {
		for i := range xs {
			xs[i] *= rad2Deg
		}
		for i := range ys {
			ys[i] *= rad2Deg
		}
	}
	return nil
}

// IsLatLong returns whether the projection uses lat/long coordinates, instead projected.
func (p *Proj) IsLatLong() bool {
	return C.pj_is_latlong(p.p) != 0
}

type Transformer struct {
	Src *Proj
	Dst *Proj
}

// Transform coordinates fron src to dst projection. Transforms coordinates in-place.
func (t *Transformer) Transform(xs, ys []float64) error {
	return t.Src.Transform(t.Dst, xs, ys)
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

var searchPaths []string
var finderResults map[string]*C.char

//export goProjFinder
func goProjFinder(cname *C.char) *C.char {
	name := C.GoString(cname)
	path, ok := finderResults[name]
	if !ok {
		for _, p := range searchPaths {
			p = filepath.Join(p, name)
			_, err := os.Stat(p)
			if err == nil {
				path = C.CString(p)
				break
			}
		}
		// cache result, even if it is nil
		finderResults[name] = path
	}
	return path
}

// SetSearchPaths add one or more directories to search for proj definition files.
// Multiple calls overwrite the previous search paths.
func SetSearchPaths(paths []string) {
	finderResults = make(map[string]*C.char)
	searchPaths = paths
	C.pj_set_finder((*[0]byte)(unsafe.Pointer(C.go_proj_finder_wrapper)))
}
