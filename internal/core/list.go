package core

import (
	"errors"

	"github.com/sayandip18/redis-clone/config"
)

var errNoSuchKey = errors.New("(error) ERR no such key")
var errIndexOutOfRange = errors.New("(error) ERR index out of range")

type listSide int

const (
	listLeft listSide = iota
	listRight
)

// NewListObj creates an empty list object encoded as ziplist.
func NewListObj() *Obj {
	return &Obj{
		Type:      ObjList,
		Encoding:  EncodingZiplist,
		Value:     newZipList(),
		ExpiresAt: -1,
		LRU:       getLRUClock(),
	}
}

func getListContainer(obj *Obj) (interface {
	Len() int
	PushLeft(string)
	PushRight(string)
	PopLeft() (string, bool)
	PopRight() (string, bool)
	Index(int) (string, bool)
	Set(int, string) bool
	Range(int, int) []string
	InsertBefore(string, string) bool
	InsertAfter(string, string) bool
	Trim(int, int)
	Rem(int, string) int
	Entries() []string
}, error) {
	if obj == nil || obj.Type != ObjList {
		return nil, ErrWrongType
	}
	switch obj.Encoding {
	case EncodingZiplist:
		z, ok := obj.Value.(*ZipList)
		if !ok {
			return nil, ErrWrongType
		}
		return z, nil
	case EncodingQuicklist:
		ql, ok := obj.Value.(*QuickList)
		if !ok {
			return nil, ErrWrongType
		}
		return ql, nil
	default:
		return nil, ErrWrongType
	}
}

func maybeConvertList(obj *Obj) {
	if obj == nil || obj.Type != ObjList || obj.Encoding != EncodingZiplist {
		return
	}
	z, ok := obj.Value.(*ZipList)
	if !ok {
		return
	}
	needConvert := z.Len() > config.ListMaxZiplistEntries
	if !needConvert && config.ListMaxZiplistValue > 0 {
		needConvert = z.maxEntryLen() > config.ListMaxZiplistValue
	}
	if !needConvert {
		return
	}
	obj.Encoding = EncodingQuicklist
	obj.Value = quicklistFromZiplist(z)
}

func listPush(key string, side listSide, values []string) (int, error) {
	obj := Get(key)
	created := false
	if obj == nil {
		obj = NewListObj()
		if err := Put(key, obj); err != nil {
			return 0, err
		}
		created = true
	} else if obj.Type != ObjList {
		return 0, ErrWrongType
	}

	c, err := getListContainer(obj)
	if err != nil {
		return 0, err
	}
	for _, v := range values {
		if side == listLeft {
			c.PushLeft(v)
		} else {
			c.PushRight(v)
		}
	}
	maybeConvertList(obj)
	if !created {
		bumpVersion(key)
	}
	tryUnblockOnPush(key)
	return c.Len(), nil
}

func listPop(key string, side listSide) (string, bool, error) {
	obj := Get(key)
	if obj == nil {
		return "", false, nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return "", false, err
	}
	var (
		v  string
		ok bool
	)
	if side == listLeft {
		v, ok = c.PopLeft()
	} else {
		v, ok = c.PopRight()
	}
	if !ok {
		return "", false, nil
	}
	if c.Len() == 0 {
		deleteKey(key)
	} else {
		bumpVersion(key)
	}
	return v, true, nil
}

func listLen(key string) (int, error) {
	obj := Get(key)
	if obj == nil {
		return 0, nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return 0, err
	}
	return c.Len(), nil
}

func listRange(key string, start, stop int) ([]string, error) {
	obj := Get(key)
	if obj == nil {
		return nil, nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return nil, err
	}
	return c.Range(start, stop), nil
}

func listIndex(key string, idx int) (string, bool, error) {
	obj := Get(key)
	if obj == nil {
		return "", false, nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return "", false, err
	}
	v, ok := c.Index(idx)
	return v, ok, nil
}

func listSet(key string, idx int, value string) error {
	obj := Get(key)
	if obj == nil {
		return errNoSuchKey
	}
	c, err := getListContainer(obj)
	if err != nil {
		return err
	}
	if !c.Set(idx, value) {
		return errIndexOutOfRange
	}
	maybeConvertList(obj)
	bumpVersion(key)
	return nil
}

func listInsert(key string, before bool, pivot, value string) (int, error) {
	obj := Get(key)
	if obj == nil {
		return -1, nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return 0, err
	}
	var ok bool
	if before {
		ok = c.InsertBefore(pivot, value)
	} else {
		ok = c.InsertAfter(pivot, value)
	}
	if !ok {
		return -1, nil
	}
	maybeConvertList(obj)
	bumpVersion(key)
	return c.Len(), nil
}

func listTrim(key string, start, stop int) error {
	obj := Get(key)
	if obj == nil {
		return nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return err
	}
	c.Trim(start, stop)
	if c.Len() == 0 {
		deleteKey(key)
	} else {
		bumpVersion(key)
	}
	return nil
}

func listRem(key string, count int, value string) (int, error) {
	obj := Get(key)
	if obj == nil {
		return 0, nil
	}
	c, err := getListContainer(obj)
	if err != nil {
		return 0, err
	}
	removed := c.Rem(count, value)
	if c.Len() == 0 {
		deleteKey(key)
	} else if removed > 0 {
		bumpVersion(key)
	}
	return removed, nil
}

// listEntries returns all elements for AOF rewrite.
func listEntries(obj *Obj) ([]string, error) {
	c, err := getListContainer(obj)
	if err != nil {
		return nil, err
	}
	return c.Entries(), nil
}
