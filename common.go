package batchgcd

import (
	"fmt"
	"github.com/ncw/gmp"
)

type Collision struct {
	Modulus *gmp.Int
	P       *gmp.Int
	Q       *gmp.Int
}

func (x Collision) HavePrivate() bool {
	return x.P != nil || x.Q != nil
}

func (x Collision) String() string {
	if x.HavePrivate() {
		if x.P.Cmp(x.Q) < 0 {
			return fmt.Sprintf("COLLISION: N=%x \nP=%x \nQ=%x", x.Modulus, x.P, x.Q)
		} else {
			return fmt.Sprintf("COLLISION: N=%x \nP=%x \nQ=%x", x.Modulus, x.Q, x.P)
		}
	} else {
		return fmt.Sprintf("DUPLICATE: %x", x.Modulus)
	}
}

func (x Collision) Test() bool {
	if !x.HavePrivate() {
		return true
	}
	n := gmp.NewInt(0)
	n.Mul(x.P, x.Q)
	return n.Cmp(x.Modulus) == 0
}

func (x Collision) Csv() string {
	if x.P.Cmp(x.Q) < 0 {
		return fmt.Sprintf("%x,%x,%x", x.Modulus, x.P, x.Q)
	} else {
		return fmt.Sprintf("%x,%x,%x", x.Modulus, x.Q, x.P)
	}
}
