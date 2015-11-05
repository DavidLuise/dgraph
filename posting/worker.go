package posting

import (
	"github.com/dgraph-io/dgraph/task"
	"github.com/google/flatbuffers/go"
)

/*
type elem struct {
	Uid   uint64
	Chidx int // channel index
}

type elemHeap []elem

func (h elemHeap) Len() int           { return len(h) }
func (h elemHeap) Less(i, j int) bool { return h[i].Uid < h[j].Uid }
func (h elemHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *elemHeap) Push(x interface{}) {
	*h = append(*h, x.(elem))
}
func (h *elemHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
*/

/*
func addUids(b *flatbuffers.Builder, sorted []uint64) flatbuffers.UOffsetT {
	// Invert the sorted uids to maintain same order in flatbuffers.
	task.ResultStartUidsVector(b, len(sorted))
	for i := len(sorted) - 1; i >= 0; i-- {
		b.PrependUint64(sorted[i])
	}
	return b.EndVector(len(sorted))
}
*/

func uidlistOffset(b *flatbuffers.Builder,
	sorted []uint64) flatbuffers.UOffsetT {

	task.UidListStartUidsVector(b, len(sorted))
	for i := len(sorted) - 1; i >= 0; i-- {
		b.PrependUint64(sorted[i])
	}
	ulist := b.EndVector(len(sorted))
	task.UidListStart(b)
	task.UidListAddUids(b, ulist)
	return task.UidListEnd(b)
}

func ProcessTask(query []byte) (result []byte, rerr error) {
	uo := flatbuffers.GetUOffsetT(query)
	q := new(task.Query)
	q.Init(query, uo)

	b := flatbuffers.NewBuilder(0)
	voffsets := make([]flatbuffers.UOffsetT, q.UidsLength())
	uoffsets := make([]flatbuffers.UOffsetT, q.UidsLength())

	attr := string(q.Attr())
	for i := 0; i < q.UidsLength(); i++ {
		uid := q.Uids(i)
		key := Key(uid, attr)
		pl := Get(key)

		task.ValueStart(b)
		var valoffset flatbuffers.UOffsetT
		if val, err := pl.Value(); err != nil {
			valoffset = b.CreateByteVector(nilbyte)
		} else {
			valoffset = b.CreateByteVector(val)
		}
		task.ValueAddVal(b, valoffset)
		voffsets[i] = task.ValueEnd(b)

		ulist := pl.GetUids()
		uoffsets[i] = uidlistOffset(b, ulist)
	}
	task.ResultStartValuesVector(b, len(voffsets))
	for i := len(voffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(voffsets[i])
	}
	valuesVent := b.EndVector(len(voffsets))

	task.ResultStartUidmatrixVector(b, len(uoffsets))
	for i := len(uoffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(uoffsets[i])
	}
	matrixVent := b.EndVector(len(uoffsets))

	task.ResultStart(b)
	task.ResultAddValues(b, valuesVent)
	task.ResultAddUidmatrix(b, matrixVent)
	rend := task.ResultEnd(b)
	b.Finish(rend)
	return b.Bytes[b.Head():], nil
}

func NewQuery(attr string, uids []uint64) []byte {
	b := flatbuffers.NewBuilder(0)
	task.QueryStartUidsVector(b, len(uids))
	for i := len(uids) - 1; i >= 0; i-- {
		b.PrependUint64(uids[i])
	}
	vend := b.EndVector(len(uids))

	ao := b.CreateString(attr)
	task.QueryStart(b)
	task.QueryAddAttr(b, ao)
	task.QueryAddUids(b, vend)
	qend := task.QueryEnd(b)
	b.Finish(qend)
	return b.Bytes[b.Head():]
}

var nilbyte []byte

func init() {
	nilbyte = make([]byte, 1)
	nilbyte[0] = 0x00
}
