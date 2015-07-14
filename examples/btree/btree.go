// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package b implements a B+tree.
//
// Changelog
//
// 2014-04-18: Added new method Put.
//
// Generic types
//
// Keys and their associated values are interface{} typed, similar to all of
// the containers in the standard library.
//
// Semiautomatic production of a type specific variant of this package is
// supported via
//
//	$ make generic
//
// This command will write to stdout a version of the btree.go file where
// every key type occurrence is replaced by the word 'key' (written in all
// CAPS) and every value type occurrence is replaced by the word 'value'
// (written in all CAPS). Then you have to replace these tokens with your
// desired type(s), using any technique you're comfortable with.
//
// This is how, for example, 'example/int.go' was created:
//
//	$ mkdir example
//	$
//	$ # Note: the command bellow must be actually written using the words
//	$ # 'key' and 'value' in all CAPS. The proper form is avoided in this
//	$ # documentation to not confuse any text replacement mechanism.
//	$
//	$ make generic | sed -e 's/key/int/g' -e 's/value/int/g' > example/int.go
//
// No other changes to int.go are (strictly) necessary, it compiles just fine.
// In a next step, benchmarks from all_test.go were copied into
// example/bench_test.go without any changes.  A comparator is defined in
// bench_test.go like this:
//
//	func cmp(a, b int) int {
//		return a - b
//	}
//
// Running the benchmarks on a machine with Intel X5450 CPU @ 3 GHz:
//
// Go release (1.1.2)
//
//	$ go test -bench . example/bench_test.go example/int.go
//	testing: warning: no tests to run
//	PASS
//	BenchmarkSetSeq	 5000000	       590 ns/op
//	BenchmarkSetRnd	 1000000	      1530 ns/op
//	BenchmarkGetSeq	10000000	       373 ns/op
//	BenchmarkGetRnd	 2000000	      1109 ns/op
//	BenchmarkDelSeq	 5000000	       672 ns/op
//	BenchmarkDelRnd	 1000000	      1275 ns/op
//	BenchmarkSeekSeq	 5000000	       552 ns/op
//	BenchmarkSeekRnd	 1000000	      1108 ns/op
//	BenchmarkNext1e3	  200000	     13414 ns/op
//	BenchmarkPrev1e3	  200000	     13215 ns/op
//	ok  	command-line-arguments	51.372s
//	$
//
// Go 1.2rc2
//
//	$ go test -bench . example/bench_test.go example/int.go
//	testing: warning: no tests to run
//	PASS
//	BenchmarkSetSeq	 5000000	       535 ns/op
//	BenchmarkSetRnd	 1000000	      1428 ns/op
//	BenchmarkGetSeq	10000000	       376 ns/op
//	BenchmarkGetRnd	 2000000	      1105 ns/op
//	BenchmarkDelSeq	 5000000	       618 ns/op
//	BenchmarkDelRnd	 1000000	      1213 ns/op
//	BenchmarkSeekSeq	 5000000	       538 ns/op
//	BenchmarkSeekRnd	 1000000	      1088 ns/op
//	BenchmarkNext1e3	  200000	     13410 ns/op
//	BenchmarkPrev1e3	  200000	     13528 ns/op
//	ok  	command-line-arguments	48.823s
//	$
//
// Note that the Next and Prev benchmarks enumerate 1000 items (KV pairs), so
// getting the next or previous iterated item is performed in about 13-14 ns.
// This is the nice O(1) property of B+trees usually not found in other tree
// types.
package btree

import (
	"io"

	"github.com/joeshaw/gengen/generic"
)

//TODO check vs orig initialize/finalize

const (
	kx = 128 //TODO benchmark tune this number if using custom key/value type(s).
	kd = 64  //TODO benchmark tune this number if using custom key/value type(s).
)

type (
	// Cmp compares a and b. Return value is:
	//
	//	< 0 if a <  b
	//	  0 if a == b
	//	> 0 if a >  b
	//
	Cmp func(a, b generic.T) int

	d struct { // data page
		c int
		d [2*kd + 1]de
		n *d
		p *d
	}

	de struct { // d element
		k generic.T
		v generic.U
	}

	// Enumerator captures the state of enumerating a tree. It is returned
	// from the Seek* methods. The enumerator is aware of any mutations
	// made to the tree in the process of enumerating it and automatically
	// resumes the enumeration at the proper key, if possible.
	//
	// However, once an Enumerator returns io.EOF to signal "no more
	// items", it does no more attempt to "resync" on tree mutation(s).  In
	// other words, io.EOF from an Enumaretor is "sticky" (idempotent).
	Enumerator struct {
		err error
		hit bool
		i   int
		k   generic.T
		q   *d
		t   *Tree
		ver int64
	}

	// Tree is a B+tree.
	Tree struct {
		c     int
		cmp   Cmp
		first *d
		last  *d
		r     interface{}
		ver   int64
	}

	xe struct { // x element
		ch  interface{}
		sep *d
	}

	x struct { // index page
		c int
		x [2*kx + 2]xe
	}
)

var ( // R/O zero values
	zd  d
	zde de
	zx  x
	zxe xe
)

func clr(q interface{}) {
	switch x := q.(type) {
	case *x:
		for i := 0; i <= x.c; i++ { // Ch0 Sep0 ... Chn-1 Sepn-1 Chn
			clr(x.x[i].ch)
		}
		*x = zx // GC
	case *d:
		*x = zd // GC
	}
}

// -------------------------------------------------------------------------- x

func newX(ch0 interface{}) *x {
	r := &x{}
	r.x[0].ch = ch0
	return r
}

func (q *x) extract(i int) {
	q.c--
	if i < q.c {
		copy(q.x[i:], q.x[i+1:q.c+1])
		q.x[q.c].ch = q.x[q.c+1].ch
		q.x[q.c].sep = nil // GC
		q.x[q.c+1] = zxe   // GC
	}
}

func (q *x) insert(i int, d *d, ch interface{}) *x {
	c := q.c
	if i < c {
		q.x[c+1].ch = q.x[c].ch
		copy(q.x[i+2:], q.x[i+1:c])
		q.x[i+1].sep = q.x[i].sep
	}
	c++
	q.c = c
	q.x[i].sep = d
	q.x[i+1].ch = ch
	return q
}

func (q *x) siblings(i int) (l, r *d) {
	if i >= 0 {
		if i > 0 {
			l = q.x[i-1].ch.(*d)
		}
		if i < q.c {
			r = q.x[i+1].ch.(*d)
		}
	}
	return
}

// -------------------------------------------------------------------------- d

func (l *d) mvL(r *d, c int) {
	copy(l.d[l.c:], r.d[:c])
	copy(r.d[:], r.d[c:r.c])
	l.c += c
	r.c -= c
}

func (l *d) mvR(r *d, c int) {
	copy(r.d[c:], r.d[:r.c])
	copy(r.d[:c], l.d[l.c-c:])
	r.c += c
	l.c -= c
}

// ----------------------------------------------------------------------- Tree

// TreeNew returns a newly created, empty Tree. The compare function is used
// for key collation.
func TreeNew(cmp Cmp) *Tree {
	return &Tree{cmp: cmp}
}

// Clear removes all K/V pairs from the tree.
func (t *Tree) Clear() {
	if t.r == nil {
		return
	}

	clr(t.r)
	t.c, t.first, t.last, t.r = 0, nil, nil, nil
	t.ver++
}

func (t *Tree) cat(p *x, q, r *d, pi int) {
	t.ver++
	q.mvL(r, r.c)
	if r.n != nil {
		r.n.p = q
	} else {
		t.last = q
	}
	q.n = r.n //TODO recycle r
	if p.c > 1 {
		p.extract(pi)
		p.x[pi].ch = q
	} else { //TODO recycle r
		t.r = q
	}
}

func (t *Tree) catX(p, q, r *x, pi int) {
	t.ver++
	q.x[q.c].sep = p.x[pi].sep
	copy(q.x[q.c+1:], r.x[:r.c])
	q.c += r.c + 1
	q.x[q.c].ch = r.x[r.c].ch //TODO recycle r
	if p.c > 1 {
		p.c--
		pc := p.c
		if pi < pc {
			p.x[pi].sep = p.x[pi+1].sep
			copy(p.x[pi+1:], p.x[pi+2:pc+1])
			p.x[pc].ch = p.x[pc+1].ch
			p.x[pc].sep = nil  // GC
			p.x[pc+1].ch = nil // GC
		}
		return
	}

	t.r = q //TODO recycle r
}

// Delete removes the k's KV pair, if it exists, in which case Delete returns
// true.
func (t *Tree) Delete(k generic.T) (ok bool) {
	pi := -1
	var p *x
	q := t.r
	if q == nil {
		return
	}

	for {
		var i int
		i, ok = t.find(q, k)
		if ok {
			switch x := q.(type) {
			case *x:
				dp := x.x[i].sep
				switch {
				case dp.c > kd:
					t.extract(dp, 0)
				default:
					if x.c < kx && q != t.r {
						t.underflowX(p, &x, pi, &i)
					}
					pi = i + 1
					p = x
					q = x.x[pi].ch
					ok = false
					continue
				}
			case *d:
				t.extract(x, i)
				if x.c >= kd {
					return
				}

				if q != t.r {
					t.underflow(p, x, pi)
				} else if t.c == 0 {
					t.Clear()
				}
			}
			return
		}

		switch x := q.(type) {
		case *x:
			if x.c < kx && q != t.r {
				t.underflowX(p, &x, pi, &i)
			}
			pi = i
			p = x
			q = x.x[i].ch
		case *d:
			return
		}
	}
}

func (t *Tree) extract(q *d, i int) { // (r generic.U) {
	t.ver++
	//r = q.d[i].v // prepared for Extract
	q.c--
	if i < q.c {
		copy(q.d[i:], q.d[i+1:q.c+1])
	}
	q.d[q.c] = zde // GC
	t.c--
	return
}

func (t *Tree) find(q interface{}, k generic.T) (i int, ok bool) {
	var mk generic.T
	l := 0
	switch x := q.(type) {
	case *x:
		h := x.c - 1
		for l <= h {
			m := (l + h) >> 1
			mk = x.x[m].sep.d[0].k
			switch cmp := t.cmp(k, mk); {
			case cmp > 0:
				l = m + 1
			case cmp == 0:
				return m, true
			default:
				h = m - 1
			}
		}
	case *d:
		h := x.c - 1
		for l <= h {
			m := (l + h) >> 1
			mk = x.d[m].k
			switch cmp := t.cmp(k, mk); {
			case cmp > 0:
				l = m + 1
			case cmp == 0:
				return m, true
			default:
				h = m - 1
			}
		}
	}
	return l, false
}

// First returns the first item of the tree in the key collating order, or
// (zero-value, zero-value) if the tree is empty.
func (t *Tree) First() (k generic.T, v generic.U) {
	if q := t.first; q != nil {
		q := &q.d[0]
		k, v = q.k, q.v
	}
	return
}

// Get returns the value associated with k and true if it exists. Otherwise Get
// returns (zero-value, false).
func (t *Tree) Get(k generic.T) (v generic.U, ok bool) {
	q := t.r
	if q == nil {
		return
	}

	for {
		var i int
		if i, ok = t.find(q, k); ok {
			switch x := q.(type) {
			case *x:
				return x.x[i].sep.d[0].v, true
			case *d:
				return x.d[i].v, true
			}
		}
		switch x := q.(type) {
		case *x:
			q = x.x[i].ch
		default:
			return
		}
	}
}

func (t *Tree) insert(q *d, i int, k generic.T, v generic.U) *d {
	t.ver++
	c := q.c
	if i < c {
		copy(q.d[i+1:], q.d[i:c])
	}
	c++
	q.c = c
	q.d[i].k, q.d[i].v = k, v
	t.c++
	return q
}

// Last returns the last item of the tree in the key collating order, or
// (zero-value, zero-value) if the tree is empty.
func (t *Tree) Last() (k generic.T, v generic.U) {
	if q := t.last; q != nil {
		q := &q.d[q.c-1]
		k, v = q.k, q.v
	}
	return
}

// Len returns the number of items in the tree.
func (t *Tree) Len() int {
	return t.c
}

func (t *Tree) overflow(p *x, q *d, pi, i int, k generic.T, v generic.U) {
	t.ver++
	l, r := p.siblings(pi)

	if l != nil && l.c < 2*kd {
		l.mvL(q, 1)
		t.insert(q, i-1, k, v)
		return
	}

	if r != nil && r.c < 2*kd {
		if i < 2*kd {
			q.mvR(r, 1)
			t.insert(q, i, k, v)
		} else {
			t.insert(r, 0, k, v)
		}
		return
	}

	t.split(p, q, pi, i, k, v)
}

// Seek returns an Enumerator positioned on a an item such that k >= item's
// key. ok reports if k == item.key The Enumerator's position is possibly
// after the last item in the tree.
func (t *Tree) Seek(k generic.T) (e *Enumerator, ok bool) {
	q := t.r
	if q == nil {
		e = &Enumerator{nil, false, 0, k, nil, t, t.ver}
		return
	}

	for {
		var i int
		if i, ok = t.find(q, k); ok {
			switch x := q.(type) {
			case *x:
				e = &Enumerator{nil, ok, 0, k, x.x[i].sep, t, t.ver}
				return
			case *d:
				e = &Enumerator{nil, ok, i, k, x, t, t.ver}
				return
			}
		}
		switch x := q.(type) {
		case *x:
			q = x.x[i].ch
		case *d:
			e = &Enumerator{nil, ok, i, k, x, t, t.ver}
			return
		}
	}
}

// SeekFirst returns an enumerator positioned on the first KV pair in the tree,
// if any. For an empty tree, err == io.EOF is returned and e will be nil.
func (t *Tree) SeekFirst() (e *Enumerator, err error) {
	q := t.first
	if q == nil {
		return nil, io.EOF
	}

	return &Enumerator{nil, true, 0, q.d[0].k, q, t, t.ver}, nil
}

// SeekLast returns an enumerator positioned on the last KV pair in the tree,
// if any. For an empty tree, err == io.EOF is returned and e will be nil.
func (t *Tree) SeekLast() (e *Enumerator, err error) {
	q := t.last
	if q == nil {
		return nil, io.EOF
	}

	return &Enumerator{nil, true, q.c - 1, q.d[q.c-1].k, q, t, t.ver}, nil
}

// Set sets the value associated with k.
func (t *Tree) Set(k generic.T, v generic.U) {
	pi := -1
	var p *x
	q := t.r
	if q != nil {
		for {
			i, ok := t.find(q, k)
			if ok {
				switch x := q.(type) {
				case *x:
					x.x[i].sep.d[0].v = v
				case *d:
					x.d[i].v = v
				}
				return
			}

			switch x := q.(type) {
			case *x:
				if x.c > 2*kx {
					t.splitX(p, &x, pi, &i)
				}
				pi = i
				p = x
				q = x.x[i].ch
			case *d:
				switch {
				case x.c < 2*kd:
					t.insert(x, i, k, v)
				default:
					t.overflow(p, x, pi, i, k, v)
				}
				return
			}
		}
	}

	z := t.insert(&d{}, 0, k, v)
	t.r, t.first, t.last = z, z, z
	return
}

// Put combines Get and Set in a more efficient way where the tree is walked
// only once. The upd(ater) receives (old-value, true) if a KV pair for k
// exists or (zero-value, false) otherwise. It can then return a (new-value,
// true) to create or overwrite the existing value in the KV pair, or
// (whatever, false) if it decides not to create or not to update the value of
// the KV pair.
//
// 	tree.Set(k, v) conceptually equals
//
// 	tree.Put(k, func(k, v []byte){ return v, true }([]byte, bool))
//
// modulo the differing return values.
func (t *Tree) Put(k generic.T, upd func(oldV generic.U, exists bool) (newV generic.U, write bool)) (oldV generic.U, written bool) {
	pi := -1
	var p *x
	q := t.r
	var newV generic.U
	if q != nil {
		for {
			i, ok := t.find(q, k)
			if ok {
				switch x := q.(type) {
				case *x:
					oldV = x.x[i].sep.d[0].v
					newV, written = upd(oldV, true)
					if !written {
						return
					}

					x.x[i].sep.d[0].v = newV
				case *d:
					oldV = x.d[i].v
					newV, written = upd(oldV, true)
					if !written {
						return
					}

					x.d[i].v = newV
				}
				return
			}

			switch x := q.(type) {
			case *x:
				if x.c > 2*kx {
					t.splitX(p, &x, pi, &i)
				}
				pi = i
				p = x
				q = x.x[i].ch
			case *d: // new KV pair
				newV, written = upd(newV, false)
				if !written {
					return
				}

				switch {
				case x.c < 2*kd:
					t.insert(x, i, k, newV)
				default:
					t.overflow(p, x, pi, i, k, newV)
				}
				return
			}
		}
	}

	// new KV pair in empty tree
	newV, written = upd(newV, false)
	if !written {
		return
	}

	z := t.insert(&d{}, 0, k, newV)
	t.r, t.first, t.last = z, z, z
	return
}

func (t *Tree) split(p *x, q *d, pi, i int, k generic.T, v generic.U) {
	t.ver++
	r := &d{}
	if q.n != nil {
		r.n = q.n
		r.n.p = r
	} else {
		t.last = r
	}
	q.n = r
	r.p = q

	copy(r.d[:], q.d[kd:2*kd])
	for i := range q.d[kd:] {
		q.d[kd+i] = zde
	}
	q.c = kd
	r.c = kd
	if pi >= 0 {
		p.insert(pi, r, r)
	} else {
		t.r = newX(q).insert(0, r, r)
	}
	if i > kd {
		t.insert(r, i-kd, k, v)
		return
	}

	t.insert(q, i, k, v)
}

func (t *Tree) splitX(p *x, pp **x, pi int, i *int) {
	t.ver++
	q := *pp
	r := &x{}
	copy(r.x[:], q.x[kx+1:])
	q.c = kx
	r.c = kx
	if pi >= 0 {
		p.insert(pi, q.x[kx].sep, r)
	} else {
		t.r = newX(q).insert(0, q.x[kx].sep, r)
	}
	q.x[kx].sep = nil
	for i := range q.x[kx+1:] {
		q.x[kx+i+1] = zxe
	}
	if *i > kx {
		*pp = r
		*i -= kx + 1
	}
}

func (t *Tree) underflow(p *x, q *d, pi int) {
	t.ver++
	l, r := p.siblings(pi)

	if l != nil && l.c+q.c >= 2*kd {
		l.mvR(q, 1)
	} else if r != nil && q.c+r.c >= 2*kd {
		q.mvL(r, 1)
		r.d[r.c] = zde // GC
	} else if l != nil {
		t.cat(p, l, q, pi-1)
	} else {
		t.cat(p, q, r, pi)
	}
}

func (t *Tree) underflowX(p *x, pp **x, pi int, i *int) {
	t.ver++
	var l, r *x
	q := *pp

	if pi >= 0 {
		if pi > 0 {
			l = p.x[pi-1].ch.(*x)
		}
		if pi < p.c {
			r = p.x[pi+1].ch.(*x)
		}
	}

	if l != nil && l.c > kx {
		q.x[q.c+1].ch = q.x[q.c].ch
		copy(q.x[1:], q.x[:q.c])
		q.x[0].ch = l.x[l.c].ch
		q.x[0].sep = p.x[pi-1].sep
		q.c++
		*i++
		l.c--
		p.x[pi-1].sep = l.x[l.c].sep
		return
	}

	if r != nil && r.c > kx {
		q.x[q.c].sep = p.x[pi].sep
		q.c++
		q.x[q.c].ch = r.x[0].ch
		p.x[pi].sep = r.x[0].sep
		copy(r.x[:], r.x[1:r.c])
		r.c--
		rc := r.c
		r.x[rc].ch = r.x[rc+1].ch
		r.x[rc].sep = nil
		r.x[rc+1].ch = nil
		return
	}

	if l != nil {
		*i += l.c + 1
		t.catX(p, l, q, pi-1)
		*pp = l
		return
	}

	t.catX(p, q, r, pi)
}

// ----------------------------------------------------------------- Enumerator

// Next returns the currently enumerated item, if it exists and moves to the
// next item in the key collation order. If there is no item to return, err ==
// io.EOF is returned.
func (e *Enumerator) Next() (k generic.T, v generic.U, err error) {
	if err = e.err; err != nil {
		return
	}

	if e.ver != e.t.ver {
		f, hit := e.t.Seek(e.k)
		if !e.hit && hit {
			if err = f.next(); err != nil {
				return
			}
		}

		*e = *f
	}
	if e.q == nil {
		e.err, err = io.EOF, io.EOF
		return
	}

	if e.i >= e.q.c {
		if err = e.next(); err != nil {
			return
		}
	}

	i := e.q.d[e.i]
	k, v = i.k, i.v
	e.k, e.hit = k, false
	e.next()
	return
}

func (e *Enumerator) next() error {
	if e.q == nil {
		e.err = io.EOF
		return io.EOF
	}

	switch {
	case e.i < e.q.c-1:
		e.i++
	default:
		if e.q, e.i = e.q.n, 0; e.q == nil {
			e.err = io.EOF
		}
	}
	return e.err
}

// Prev returns the currently enumerated item, if it exists and moves to the
// previous item in the key collation order. If there is no item to return, err
// == io.EOF is returned.
func (e *Enumerator) Prev() (k generic.T, v generic.U, err error) {
	if err = e.err; err != nil {
		return
	}

	if e.ver != e.t.ver {
		f, hit := e.t.Seek(e.k)
		if !e.hit && hit {
			if err = f.prev(); err != nil {
				return
			}
		}

		*e = *f
	}
	if e.q == nil {
		e.err, err = io.EOF, io.EOF
		return
	}

	if e.i >= e.q.c {
		if err = e.next(); err != nil {
			return
		}
	}

	i := e.q.d[e.i]
	k, v = i.k, i.v
	e.k, e.hit = k, false
	e.prev()
	return
}

func (e *Enumerator) prev() error {
	if e.q == nil {
		e.err = io.EOF
		return io.EOF
	}

	switch {
	case e.i > 0:
		e.i--
	default:
		if e.q = e.q.p; e.q == nil {
			e.err = io.EOF
			break
		}

		e.i = e.q.c - 1
	}
	return e.err
}
