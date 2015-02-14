package batchgcd

import (
	"math/big"
	"sync"
)

// This performs the GCD of the product of all previous moduli with
// the current one.  This uses around double the memory (minus quite a lot of overhead)
// If we get a GCD that is the same as the modulus, we do a manual scan for either colliding Q or identical moduli
// If we get a GCD lower than the modulus, we have one private key, then do a manual scan for others.
func MulAccumGCD(moduli []big.Int, collisions chan<- Collision) {
	accum := big.NewInt(1)
	gcd := &big.Int{}
	var wg sync.WaitGroup

	for i := 0; i < len(moduli); i++ {
		n := &moduli[i]
		gcd.GCD(nil, nil, accum, n)
		if gcd.BitLen() != 1 {
			wg.Add(1)
			if gcd.Cmp(n) == 0 {
				go findGCD(&wg, moduli, i, collisions)
				continue
			} else {
				go findDivisors(&wg, moduli, i, gcd, collisions)
				gcd = &big.Int{}
			}
		}
		accum.Mul(accum, n)
	}
	wg.Wait()
	close(collisions)
}

func findDivisors(wg *sync.WaitGroup, moduli []big.Int, i int, gcd *big.Int, collisions chan<- Collision) {
	m := &moduli[i]
	q := &big.Int{}

	q.Quo(m, gcd)
	collisions <- Collision{
		Modulus: m,
		P:       gcd,
		Q:       q,
	}
	q = &big.Int{}

	for j := 0; j < len(moduli); j++ {
		if j == i {
			continue
		}
		n := &moduli[j]
		if n.Cmp(m) == 0 {
			collisions <- Collision{Modulus: m}
		} else if q.Rem(n, gcd).BitLen() == 0 {
			q.Quo(n, gcd)
			collisions <- Collision{
				Modulus: n,
				P:       gcd,
				Q:       q,
			}
			q = &big.Int{}
		}
	}
	wg.Done()
}

func findGCD(wg *sync.WaitGroup, moduli []big.Int, i int, collisions chan<- Collision) {
	m := &moduli[i]
	q := &big.Int{}
	gcd := &big.Int{}

	for j := 0; j < len(moduli); j++ {
		if j == i {
			break
		}
		n := &moduli[j]

		if n.Cmp(m) == 0 {
			collisions <- Collision{Modulus: m}
		} else if gcd.GCD(nil, nil, m, n).BitLen() != 1 {
			q.Quo(m, gcd)
			collisions <- Collision{
				Modulus: m,
				P:       gcd,
				Q:       q,
			}
			q = &big.Int{}

			q.Quo(n, gcd)
			collisions <- Collision{
				Modulus: n,
				P:       gcd,
				Q:       q,
			}
			q = &big.Int{}

			gcd = &big.Int{}
		}
	}
	wg.Done()
}
