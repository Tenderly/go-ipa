package bls

func (p *G1Point) String() string {
	return StrG1(p)
}

func (p *G2Point) String() string {
	return StrG2(p)
}
