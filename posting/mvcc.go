/*
 * Copyright 2017-2018 Dgraph Labs, Inc.
 *
 * This file is available under the Apache License, Version 2.0,
 * with the Commons Clause restriction.
 */

package posting

import (
	"bytes"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/dgraph-io/dgraph/protos/pb"
	"github.com/dgraph-io/dgraph/x"
	farm "github.com/dgryski/go-farm"
)

var (
	ErrTsTooOld = x.Errorf("Transaction is too old")
)

func (t *Txn) SetAbort() {
	atomic.StoreUint32(&t.shouldAbort, 1)
}

func (t *Txn) ShouldAbort() bool {
	if t == nil {
		return false
	}
	return atomic.LoadUint32(&t.shouldAbort) > 0
}

func (t *Txn) AddKeys(key, conflictKey string) {
	t.Lock()
	defer t.Unlock()
	if t.deltas == nil || t.conflicts == nil {
		t.deltas = make(map[string]struct{})
		t.conflicts = make(map[string]struct{})
	}
	t.deltas[key] = struct{}{}
	if len(conflictKey) > 0 {
		t.conflicts[conflictKey] = struct{}{}
	}
}

func (t *Txn) Fill(ctx *api.TxnContext) {
	t.Lock()
	defer t.Unlock()
	ctx.StartTs = t.StartTs
	for key := range t.conflicts {
		// We don't need to send the whole conflict key to Zero. Solving #2338
		// should be done by sending a list of mutating predicates to Zero,
		// along with the keys to be used for conflict detection.
		fps := strconv.FormatUint(farm.Fingerprint64([]byte(key)), 36)
		if !x.HasString(ctx.Keys, fps) {
			ctx.Keys = append(ctx.Keys, fps)
		}
	}
	for key := range t.deltas {
		pk := x.Parse([]byte(key))
		if !x.HasString(ctx.Preds, pk.Attr) {
			ctx.Preds = append(ctx.Preds, pk.Attr)
		}
	}
}

// Don't call this for schema mutations. Directly commit them.
// This function only stores deltas to the commit timestamps. It does not try to generate a state.
// TODO: Simplify this function. All it should be doing is to store the deltas, and not try to
// generate state. The state should only be generated by rollup, which in turn should look at the
// last Snapshot Ts, to determine how much of the PL to rollup. We only want to roll up the deltas,
// with commit ts <= snapshot ts, and not above.
func (tx *Txn) CommitToDisk(writer *x.TxnWriter, commitTs uint64) error {
	if commitTs == 0 {
		return nil
	}
	var keys []string
	tx.Lock()
	for key := range tx.deltas {
		keys = append(keys, key)
	}
	tx.Unlock()

	// TODO: Simplify this. All we need to do is to get the PL for the key, and if it has the
	// postings for the startTs, we commit them. Otherwise, we skip.
	// Also, if the snapshot read ts is above the commit ts, then we just delete the postings from
	// memory, instead of writing them back again.

	for _, key := range keys {
		plist, err := Get([]byte(key))
		if err != nil {
			return err
		}
		data := plist.GetMutation(tx.StartTs)
		if data == nil {
			continue
		}
		if err := writer.SetAt([]byte(key), data, bitDeltaPosting, commitTs); err != nil {
			return err
		}
	}
	return nil
}

func (tx *Txn) CommitToMemory(commitTs uint64) error {
	tx.Lock()
	defer tx.Unlock()
	// TODO: Figure out what shouldAbort is for, and use it correctly. This should really be
	// shouldDiscard.
	// defer func() {
	// 	atomic.StoreUint32(&tx.shouldAbort, 1)
	// }()
	for key := range tx.deltas {
		for {
			plist, err := Get([]byte(key))
			if err != nil {
				return err
			}
			err = plist.CommitMutation(tx.StartTs, commitTs)
			if err == ErrRetry {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			if err == nil {
				break
			}
			x.Errorf("While commiting to memory: %v\n", err)
			return err
		}
	}
	return nil
}

func unmarshalOrCopy(plist *pb.PostingList, item *badger.Item) error {
	// It's delta
	return item.Value(func(val []byte) error {
		if len(val) == 0 {
			// empty pl
			return nil
		}
		// Found complete pl, no needn't iterate more
		if item.UserMeta()&BitUidPosting != 0 {
			plist.Uids = make([]byte, len(val))
			copy(plist.Uids, val)
		} else if len(val) > 0 {
			x.Check(plist.Unmarshal(val))
		}
		return nil
	})
}

// constructs the posting list from the disk using the passed iterator.
// Use forward iterator with allversions enabled in iter options.
//
// key would now be owned by the posting list. So, ensure that it isn't reused
// elsewhere.
func ReadPostingList(key []byte, it *badger.Iterator) (*List, error) {
	l := new(List)
	l.key = key
	l.mutationMap = make(map[uint64]*pb.PostingList)
	l.plist = new(pb.PostingList)

	// Iterates from highest Ts to lowest Ts
	for it.Valid() {
		item := it.Item()
		if item.IsDeletedOrExpired() {
			// Don't consider any more versions.
			break
		}
		if !bytes.Equal(item.Key(), l.key) {
			break
		}
		if l.commitTs == 0 {
			l.commitTs = item.Version()
		}

		if item.UserMeta()&BitCompletePosting > 0 {
			if err := unmarshalOrCopy(l.plist, item); err != nil {
				return nil, err
			}
			l.minTs = item.Version()
			// No need to do Next here. The outer loop can take care of skipping more versions of
			// the same key.
			break
		}

		if item.UserMeta()&bitDeltaPosting > 0 {
			err := item.Value(func(val []byte) error {
				pl := &pb.PostingList{}
				x.Check(pl.Unmarshal(val))
				pl.Commit = item.Version()
				for _, mpost := range pl.Postings {
					// commitTs, startTs are meant to be only in memory, not
					// stored on disk.
					mpost.CommitTs = item.Version()
				}
				l.mutationMap[pl.Commit] = pl
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			x.Fatalf("unexpected meta: %d", item.UserMeta())
		}
		if item.DiscardEarlierVersions() {
			break
		}
		it.Next()
	}
	return l, nil
}

func getNew(key []byte, pstore *badger.DB) (*List, error) {
	l := new(List)
	l.key = key
	l.mutationMap = make(map[uint64]*pb.PostingList)
	l.plist = new(pb.PostingList)
	txn := pstore.NewTransactionAt(math.MaxUint64, false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return l, nil
	}
	if err != nil {
		return l, err
	}
	if item.UserMeta()&BitCompletePosting > 0 {
		err = unmarshalOrCopy(l.plist, item)
		l.minTs = item.Version()
		l.commitTs = item.Version()
	} else {
		iterOpts := badger.DefaultIteratorOptions
		iterOpts.AllVersions = true
		it := txn.NewIterator(iterOpts)
		defer it.Close()
		it.Seek(key)
		l, err = ReadPostingList(key, it)
	}

	if err != nil {
		return l, err
	}

	l.onDisk = 1
	l.Lock()
	size := l.calculateSize()
	l.Unlock()
	x.BytesRead.Add(int64(size))
	atomic.StoreInt32(&l.estimatedSize, size)
	return l, nil
}

type BTreeIterator struct {
	keys    [][]byte
	idx     int
	Reverse bool
	Prefix  []byte
}

func (bi *BTreeIterator) Next() {
	bi.idx++
}

func (bi *BTreeIterator) Key() []byte {
	x.AssertTrue(bi.Valid())
	return bi.keys[bi.idx]
}

func (bi *BTreeIterator) Valid() bool {
	// No need to check HasPrefix here, because we are only picking those keys
	// which have the right prefix in Seek.
	return bi.idx < len(bi.keys)
}

func (bi *BTreeIterator) Seek(key []byte) {
	bi.keys = bi.keys[:0]
	bi.idx = 0
	cont := func(key []byte) bool {
		if !bytes.HasPrefix(key, bi.Prefix) {
			return false
		}
		bi.keys = append(bi.keys, key)
		return true
	}
	if !bi.Reverse {
		btree.AscendGreaterOrEqual(key, cont)
	} else {
		btree.DescendLessOrEqual(key, cont)
	}
}

type TxnPrefixIterator struct {
	btreeIter  *BTreeIterator
	badgerIter *badger.Iterator
	prefix     []byte
	reverse    bool
	curKey     []byte
	userMeta   byte // userMeta stored as part of badger item, used to skip empty PL in has query.
}

func NewTxnPrefixIterator(txn *badger.Txn,
	iterOpts badger.IteratorOptions, prefix, key []byte) *TxnPrefixIterator {
	x.AssertTrue(iterOpts.PrefetchValues == false)
	txnIt := new(TxnPrefixIterator)
	txnIt.reverse = iterOpts.Reverse
	txnIt.prefix = prefix
	txnIt.btreeIter = &BTreeIterator{
		Reverse: iterOpts.Reverse,
		Prefix:  prefix,
	}
	txnIt.btreeIter.Seek(key)
	// Create iterator only after copying the keys from btree, or else there could
	// be race after creating iterator and before reading btree. Some keys might end up
	// getting deleted and iterator won't be initialized with new memtbales.
	txnIt.badgerIter = txn.NewIterator(iterOpts)
	txnIt.badgerIter.Seek(key)
	txnIt.Next()
	return txnIt
}

func (t *TxnPrefixIterator) Valid() bool {
	return len(t.curKey) > 0
}

func (t *TxnPrefixIterator) compare(key1 []byte, key2 []byte) int {
	if !t.reverse {
		return bytes.Compare(key1, key2)
	}
	return bytes.Compare(key2, key1)
}

func (t *TxnPrefixIterator) Next() {
	if len(t.curKey) > 0 {
		// Avoid duplicate keys during merging.
		for t.btreeIter.Valid() && t.compare(t.btreeIter.Key(), t.curKey) <= 0 {
			t.btreeIter.Next()
		}
		for t.badgerIter.ValidForPrefix(t.prefix) &&
			t.compare(t.badgerIter.Item().Key(), t.curKey) <= 0 {
			t.badgerIter.Next()
		}
	}

	t.userMeta = 0 // reset it.
	if !t.btreeIter.Valid() && !t.badgerIter.ValidForPrefix(t.prefix) {
		t.curKey = nil
		return
	} else if !t.badgerIter.ValidForPrefix(t.prefix) {
		t.storeKey(t.btreeIter.Key())
		t.btreeIter.Next()
	} else if !t.btreeIter.Valid() {
		t.userMeta = t.badgerIter.Item().UserMeta()
		t.storeKey(t.badgerIter.Item().Key())
		t.badgerIter.Next()
	} else { // Both are valid
		if t.compare(t.btreeIter.Key(), t.badgerIter.Item().Key()) < 0 {
			t.storeKey(t.btreeIter.Key())
			t.btreeIter.Next()
		} else {
			t.userMeta = t.badgerIter.Item().UserMeta()
			t.storeKey(t.badgerIter.Item().Key())
			t.badgerIter.Next()
		}
	}
}

func (t *TxnPrefixIterator) UserMeta() byte {
	return t.userMeta
}

func (t *TxnPrefixIterator) storeKey(key []byte) {
	if cap(t.curKey) < len(key) {
		t.curKey = make([]byte, 2*len(key))
	}
	t.curKey = t.curKey[:len(key)]
	copy(t.curKey, key)
}

func (t *TxnPrefixIterator) Key() []byte {
	return t.curKey
}

func (t *TxnPrefixIterator) Close() {
	t.badgerIter.Close()
}
