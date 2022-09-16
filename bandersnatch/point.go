// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT
// This file has been editted to fix bugs. In particular, Bytes, ScalarMul, PointAddition(Proj)

package bandersnatch

import (
	"crypto/subtle"

	"io"

	"github.com/crate-crypto/go-ipa/bandersnatch/fp"
	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
)

// PointAffine point on a twisted Edwards curve
type PointAffine struct {
	X, Y fp.Element
}

// PointProj point in projective coordinates
type PointProj struct {
	X, Y, Z fp.Element
}

const (
	//following https://tools.ietf.org/html/rfc8032#section-3.1,
	// an fr element x is negative if its binary encoding is
	// lexicographically larger than -x.
	mCompressedNegative = 0x80
	mCompressedPositive = 0x00
	mUnmask             = 0x7f

	// size in byte of a compressed point (point.Y --> fp.Element)
	sizePointCompressed = fp.Limbs * 8
)

// Bytes returns the compressed point as a byte array
// Follows https://tools.ietf.org/html/rfc8032#section-3.1,
// as the twisted Edwards implementation is primarily used
// for eddsa.
func (p *PointAffine) Bytes() [sizePointCompressed]byte {

	var res [sizePointCompressed]byte
	var mask uint

	y := p.X.Bytes()

	if p.Y.LexicographicallyLargest() {
		mask = mCompressedNegative
	} else {
		mask = mCompressedPositive
	}
	// p.Y must be in little endian
	y[0] |= byte(mask) // msb of y
	for i, j := 0, sizePointCompressed-1; i < j; i, j = i+1, j-1 {
		y[i], y[j] = y[j], y[i]
	}
	subtle.ConstantTimeCopy(1, res[:], y[:])
	return res
}

// Marshal converts p to a byte slice
func (p *PointAffine) Marshal() []byte {
	b := p.Bytes()
	return b[:]
}

// SetBytes sets p from buf
// len(buf) >= sizePointCompressed
// buf contains the X coordinate masked with a parity bit to recompute the Y coordinate
// from the curve equation. See Bytes() and https://tools.ietf.org/html/rfc8032#section-3.1
// Returns the number of read bytes and an error if the buffer is too short.
func (p *PointAffine) SetBytes(buf []byte) (int, error) {

	if len(buf) < sizePointCompressed {
		return 0, io.ErrShortBuffer
	}
	bufCopy := make([]byte, sizePointCompressed)
	subtle.ConstantTimeCopy(1, bufCopy, buf[:sizePointCompressed])
	for i, j := 0, sizePointCompressed-1; i < j; i, j = i+1, j-1 {
		bufCopy[i], bufCopy[j] = bufCopy[j], bufCopy[i]
	}
	isLexicographicallyLargest := (mCompressedNegative&bufCopy[0])>>7 == 1
	bufCopy[0] &= mUnmask
	p.X.SetBytes(bufCopy)
	p.Y = *computeY(&p.X, isLexicographicallyLargest)

	return sizePointCompressed, nil
}

// Reads an uncompressed affine point
// Point is not guaranteed to be in the prime subgroup 
func ReadUncompressedPoint(r io.Reader) PointAffine {
	var xy = make([]byte, 64)
	n, err := r.Read(xy[:32])
	if err != nil {
		panic("error reading bytes")
	}
	if n != 32 {
		panic("did not read enough bytes")
	}
	n, err = r.Read(xy[32:])
	if err != nil {
		panic("error reading bytes")
	}
	if n != 32 {
		panic("did not read enough bytes")
	}

	var x_fp = fp.Element{}
	x_fp.SetBytes(xy[:32])
	var y_fp = fp.Element{}
	y_fp.SetBytes(xy[32:])

	return PointAffine{
		X: x_fp,
		Y: y_fp,
	}
}
// Writes an uncompressed affine point to an io.Writer
func (p *PointAffine) WriteUncompressedPoint(w io.Writer) (int, error) {
	x_bytes := p.X.Bytes()
	y_bytes := p.Y.Bytes()
	n1, err := w.Write(x_bytes[:])
	if err != nil {
		return n1, err
	}
	n2, err := w.Write(y_bytes[:])
	total_bytes_written := n1 + n2
	if err != nil {
		return total_bytes_written, err
	}
	return total_bytes_written, nil
}


// Unmarshal alias to SetBytes()
func (p *PointAffine) Unmarshal(b []byte) error {
	_, err := p.SetBytes(b)
	return err
}

// Set sets p to p1 and return it
func (p *PointProj) Set(p1 *PointProj) *PointProj {
	p.X.Set(&p1.X)
	p.Y.Set(&p1.Y)
	p.Z.Set(&p1.Z)
	return p
}

// Set sets p to p1 and return it
func (p *PointAffine) Set(p1 *PointAffine) *PointAffine {
	p.X.Set(&p1.X)
	p.Y.Set(&p1.Y)
	return p
}

// Set sets p to be the identity point
func (p *PointAffine) Identity() *PointAffine {

	p.X.SetZero()
	p.Y.SetOne()

	return p
}

// Set sets p to be the identity point
func (p *PointProj) Identity() *PointProj {

	p.X.SetZero()
	p.Y.SetOne()
	p.Z.SetOne()

	return p
}

// Equal returns true if p=p1 false otherwise
func (p *PointAffine) Equal(p1 *PointAffine) bool {
	return p.X.Equal(&p1.X) && p.Y.Equal(&p1.Y)
}

// Equal returns true if p=p1 false otherwise
// If one point is on the affine chart Z=0 it returns false
func (p *PointProj) Equal(p1 *PointProj) bool {
	if p.Z.IsZero() || p1.Z.IsZero() {
		return false
	}
	var pAffine, p1Affine PointAffine
	pAffine.FromProj(p)
	p1Affine.FromProj(p1)
	return pAffine.Equal(&p1Affine)
}

// NewPointAffine creates a new instance of PointAffine
func NewPointAffine(x, y fp.Element) PointAffine {
	return PointAffine{x, y}
}

// IsOnCurve checks if a point is on the twisted Edwards curve
func (p *PointAffine) IsOnCurve() bool {

	var lhs, rhs, tmp fp.Element

	tmp.Mul(&p.Y, &p.Y)
	lhs.Mul(&p.X, &p.X).
		Mul(&lhs, &edwards.A).
		Add(&lhs, &tmp)

	tmp.Mul(&p.X, &p.X).
		Mul(&tmp, &p.Y).
		Mul(&tmp, &p.Y).
		Mul(&tmp, &edwards.D)
	rhs.SetOne().Add(&rhs, &tmp)

	return lhs.Equal(&rhs)
}

// Add adds two points (x,y), (u,v) on a twisted Edwards curve with parameters a, d
// modifies p
func (p *PointAffine) Add(p1, p2 *PointAffine) *PointAffine {

	var xu, yv, xv, yu, dxyuv, one, denx, deny fp.Element
	pRes := new(PointAffine)
	xv.Mul(&p1.X, &p2.Y)
	yu.Mul(&p1.Y, &p2.X)
	pRes.X.Add(&xv, &yu)

	xu.Mul(&p1.X, &p2.X).Mul(&xu, &edwards.A)
	yv.Mul(&p1.Y, &p2.Y)
	pRes.Y.Sub(&yv, &xu)

	dxyuv.Mul(&xv, &yu).Mul(&dxyuv, &edwards.D)
	one.SetOne()
	denx.Add(&one, &dxyuv)
	deny.Sub(&one, &dxyuv)

	p.X.Div(&pRes.X, &denx)
	p.Y.Div(&pRes.Y, &deny)

	return p
}
func (p *PointAffine) Sub(p1, p2 *PointAffine) *PointAffine {
	var neg_p2 PointAffine
	neg_p2.Neg(p2)
	return p.Add(p1, &neg_p2)
}

// Double doubles point (x,y) on a twisted Edwards curve with parameters a, d
// modifies p
func (p *PointAffine) Double(p1 *PointAffine) *PointAffine {
	p.Add(p1, p1)
	return p
}

// Neg negates point (x,y) on a twisted Edwards curve with parameters a, d
// modifies p
func (p *PointAffine) Neg(p1 *PointAffine) *PointAffine {
	p.Set(p1)
	p.X.Neg(&p1.X)
	return p
}

// FromProj sets p in affine from p in projective
func (p *PointAffine) FromProj(p1 *PointProj) *PointAffine {
	var one = fp.One()

	if p1.Z.Equal(&one) {
		p.X.Set(&p1.X)
		p.Y.Set(&p1.Y)
		return p
	}
	
	var zInv fp.Element
	zInv.Inverse(&p1.Z)
	
	p.X.Mul(&p1.X, &zInv)
	p.Y.Mul(&p1.Y, &zInv)
	return p
}

// FromAffine sets p in projective from p in affine
func (p *PointProj) FromAffine(p1 *PointAffine) *PointProj {
	p.X.Set(&p1.X)
	p.Y.Set(&p1.Y)
	p.Z.SetOne()
	return p
}

// Add adds points in projective coordinates
// cf https://hyperelliptic.org/EFD/g1p/auto-twisted-projective.html
func (p *PointProj) Add(p1, p2 *PointProj) *PointProj {

	var res PointProj

	var A, B, C, D, E, F, G, H, I fp.Element
	A.Mul(&p1.Z, &p2.Z)
	B.Square(&A)
	C.Mul(&p1.X, &p2.X)
	D.Mul(&p1.Y, &p2.Y)
	E.Mul(&edwards.D, &C).Mul(&E, &D)
	F.Sub(&B, &E)
	G.Add(&B, &E)
	H.Add(&p1.X, &p1.Y)
	I.Add(&p2.X, &p2.Y)
	res.X.Mul(&H, &I).
		Sub(&res.X, &C).
		Sub(&res.X, &D).
		Mul(&res.X, &A).
		Mul(&res.X, &F)
	H.Mul(&edwards.A, &C)
	res.Y.Sub(&D, &H).
		Mul(&res.Y, &A).
		Mul(&res.Y, &G)
	res.Z.Mul(&F, &G)

	p.Set(&res)
	return p
}

// Double adds points in projective coordinates
// cf https://hyperelliptic.org/EFD/g1p/auto-twisted-projective.html
func (p *PointProj) Double(p1 *PointProj) *PointProj {

	var res PointProj

	var B, C, D, E, F, H, J, tmp fp.Element

	B.Add(&p1.X, &p1.Y).Square(&B)
	C.Square(&p1.X)
	D.Square(&p1.Y)
	E.Mul(&edwards.A, &C)
	F.Add(&E, &D)
	H.Square(&p1.Z)
	tmp.Double(&H)
	J.Sub(&F, &tmp)
	res.X.Sub(&B, &C).
		Sub(&res.X, &D).
		Mul(&res.X, &J)
	res.Y.Sub(&E, &D).Mul(&res.Y, &F)
	res.Z.Mul(&F, &J)

	p.Set(&res)
	return p
}

// Neg sets p to -p1 and returns it
func (p *PointProj) Neg(p1 *PointProj) *PointProj {
	p.Set(p1)
	p.X.Neg(&p1.X)
	return p
}

func (p *PointAffine) ScalarMul(p1 *PointAffine, scalar_mont *fr.Element) *PointAffine {

	var resProj, p1Proj PointProj
	resProj.Identity()
	p1Proj.FromAffine(p1)

	scalar := scalar_mont.ToRegular()
	bit_len := scalar.BitLen()

	for i := bit_len; i >= 0; i-- {
		resProj.Double(&resProj)
		if scalar.Bit(uint64(i)) == 1 {
			resProj.Add(&resProj, &p1Proj)
		}
	}

	p.FromProj(&resProj)

	return p

}
func (p *PointProj) ScalarMul(p1 *PointProj, scalar_mont *fr.Element) *PointProj {

	var resProj, p1Proj PointProj
	resProj.Identity()
	p1Proj.Set(p1)

	scalar := scalar_mont.ToRegular()
	bit_len := scalar.BitLen()

	for i := bit_len; i >= 0; i-- {
		resProj.Double(&resProj)
		if scalar.Bit(uint64(i)) == 1 {
			resProj.Add(&resProj, &p1Proj)
		}
	}

	p.Set(&resProj)

	return p
}

// All points in the prime subgroup have prime order
// so we can check for prime order by multiplying by the order
func (p PointAffine) IsInPrimeSubgroup() bool {

	var order = GetEdwardsCurve().Order

	var resProj, p1Proj PointProj
	resProj.Identity()
	p1Proj.FromAffine(&p)

	bit_len := order.BitLen()

	for i := bit_len; i >= 0; i-- {
		resProj.Double(&resProj)
		if order.Bit(i) == 1 {
			resProj.Add(&resProj, &p1Proj)
		}
	}

	var tmp PointAffine
	tmp.FromProj(&resProj)

	var identity PointAffine
	identity.Identity()

	return identity.Equal(&tmp)
}

func GetPointFromX(x *fp.Element, choose_largest bool) *PointAffine {

	y := computeY(x, choose_largest)
	if y == nil { // not a square
		return nil
	}
	return &PointAffine{X: *x, Y: *y}
}

// ax^2 + y^2 = 1 + dx^2y^2
// ax^2 -1 = dx^2y^2 - y^2
// ax^2 -1 = y^2(dx^2 -1)
// ax^2 - 1 / (dx^2 - 1) = y^2
func computeY(x *fp.Element, choose_largest bool) *fp.Element {

	var one, num, den, y fp.Element
	one.SetOne()
	num.Square(x)       // x^2
	den.Mul(&num, &edwards.D)   //dx^2
	den.Sub(&den, &one) //dx^2 - 1

	num.Mul(&num, &edwards.A)   // ax^2
	num.Sub(&num, &one) // ax^2 - 1
	y.Div(&num, &den)
	is_nil := y.Sqrt(&y)

	// If the square root does not exist, then the Sqrt method returns nil
	// and leaves the receiver unchanged.
	// Note the fact that it leaves the receiver unchanged, means we do not return &y
	if is_nil == nil {
		return nil
	}

	// Choose between `y` and it's negation
	is_largest := y.LexicographicallyLargest()
	if choose_largest == is_largest {
		return &y
	} else {
		return y.Neg(&y)
	}

}
