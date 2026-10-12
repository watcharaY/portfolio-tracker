package domain

type Money int64

type BasisPoints int64

func (bp BasisPoints) Apply(m Money) Money {
	return Money(int64(m) * int64(bp) / 10_000)
}

func Ratio(num, den Money) *float64 {
	if den == 0 {
		return nil
	}
	r := float64(num) / float64(den)
	return &r
}
