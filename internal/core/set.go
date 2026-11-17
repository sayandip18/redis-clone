package core

import (
	"math/rand"

	"github.com/sayandip18/redis-clone/config"
)

// NewSetObj creates an empty set. encodingHint chooses initial encoding when first member is known.
func NewSetObj() *Obj {
	return &Obj{
		Type:      ObjSet,
		Encoding:  EncodingIntset,
		Value:     newIntSet(),
		ExpiresAt: -1,
		LRU:       getLRUClock(),
	}
}

func maybeConvertSet(obj *Obj) {
	if obj == nil || obj.Type != ObjSet || obj.Encoding != EncodingIntset {
		return
	}
	is, ok := obj.Value.(*IntSet)
	if !ok {
		return
	}
	if is.Len() > config.SetMaxIntsetEntries {
		obj.Encoding = EncodingHT
		obj.Value = is.toHashtable()
	}
}

func convertSetToHT(obj *Obj) {
	if obj.Encoding == EncodingHT {
		return
	}
	is, ok := obj.Value.(*IntSet)
	if !ok {
		obj.Encoding = EncodingHT
		obj.Value = make(map[string]struct{})
		return
	}
	obj.Encoding = EncodingHT
	obj.Value = is.toHashtable()
}

func setAdd(key string, members []string) (int, error) {
	obj := Get(key)
	created := false
	if obj == nil {
		// Prefer intset if all members are integers.
		allInt := true
		for _, m := range members {
			if _, ok := parseIntMember(m); !ok {
				allInt = false
				break
			}
		}
		obj = NewSetObj()
		if !allInt {
			obj.Encoding = EncodingHT
			obj.Value = make(map[string]struct{})
		}
		if err := Put(key, obj); err != nil {
			return 0, err
		}
		created = true
	} else if obj.Type != ObjSet {
		return 0, ErrWrongType
	}

	added := 0
	for _, m := range members {
		n, isInt := parseIntMember(m)
		if obj.Encoding == EncodingIntset {
			if !isInt {
				convertSetToHT(obj)
			} else {
				is := obj.Value.(*IntSet)
				if is.Add(n) {
					added++
				}
				maybeConvertSet(obj)
				continue
			}
		}
		ht := obj.Value.(map[string]struct{})
		if _, exists := ht[m]; !exists {
			ht[m] = struct{}{}
			added++
		}
	}
	if !created && added > 0 {
		bumpVersion(key)
	}
	return added, nil
}

func setRem(key string, members []string) (int, error) {
	obj := Get(key)
	if obj == nil {
		return 0, nil
	}
	if obj.Type != ObjSet {
		return 0, ErrWrongType
	}
	removed := 0
	switch obj.Encoding {
	case EncodingIntset:
		is := obj.Value.(*IntSet)
		for _, m := range members {
			if n, ok := parseIntMember(m); ok && is.Remove(n) {
				removed++
			}
		}
		if is.Len() == 0 {
			deleteKey(key)
			return removed, nil
		}
	case EncodingHT:
		ht := obj.Value.(map[string]struct{})
		for _, m := range members {
			if _, ok := ht[m]; ok {
				delete(ht, m)
				removed++
			}
		}
		if len(ht) == 0 {
			deleteKey(key)
			return removed, nil
		}
	default:
		return 0, ErrWrongType
	}
	if removed > 0 {
		bumpVersion(key)
	}
	return removed, nil
}

func setIsMember(key, member string) (bool, error) {
	obj := Get(key)
	if obj == nil {
		return false, nil
	}
	if obj.Type != ObjSet {
		return false, ErrWrongType
	}
	switch obj.Encoding {
	case EncodingIntset:
		n, ok := parseIntMember(member)
		if !ok {
			return false, nil
		}
		return obj.Value.(*IntSet).Contains(n), nil
	case EncodingHT:
		_, ok := obj.Value.(map[string]struct{})[member]
		return ok, nil
	default:
		return false, ErrWrongType
	}
}

func setCard(key string) (int, error) {
	obj := Get(key)
	if obj == nil {
		return 0, nil
	}
	if obj.Type != ObjSet {
		return 0, ErrWrongType
	}
	switch obj.Encoding {
	case EncodingIntset:
		return obj.Value.(*IntSet).Len(), nil
	case EncodingHT:
		return len(obj.Value.(map[string]struct{})), nil
	default:
		return 0, ErrWrongType
	}
}

func setMembers(key string) ([]string, error) {
	obj := Get(key)
	if obj == nil {
		return nil, nil
	}
	return setMembersObj(obj)
}

func setMembersObj(obj *Obj) ([]string, error) {
	if obj.Type != ObjSet {
		return nil, ErrWrongType
	}
	switch obj.Encoding {
	case EncodingIntset:
		return obj.Value.(*IntSet).Members(), nil
	case EncodingHT:
		ht := obj.Value.(map[string]struct{})
		out := make([]string, 0, len(ht))
		for m := range ht {
			out = append(out, m)
		}
		return out, nil
	default:
		return nil, ErrWrongType
	}
}

func setRandom(key string, count int, withCount bool) ([]string, error) {
	obj := Get(key)
	if obj == nil {
		if withCount {
			return []string{}, nil
		}
		return nil, nil
	}
	members, err := setMembersObj(obj)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		if withCount {
			return []string{}, nil
		}
		return nil, nil
	}
	if !withCount {
		return []string{members[rand.Intn(len(members))]}, nil
	}
	if count >= 0 {
		if count > len(members) {
			count = len(members)
		}
		rand.Shuffle(len(members), func(i, j int) { members[i], members[j] = members[j], members[i] })
		return members[:count], nil
	}
	// Negative count: allow duplicates, return |count| elements
	n := -count
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = members[rand.Intn(len(members))]
	}
	return out, nil
}

func setPop(key string, count int, withCount bool) ([]string, error) {
	obj := Get(key)
	if obj == nil {
		if withCount {
			return []string{}, nil
		}
		return nil, nil
	}
	if obj.Type != ObjSet {
		return nil, ErrWrongType
	}
	members, err := setMembersObj(obj)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		if withCount {
			return []string{}, nil
		}
		return nil, nil
	}

	n := 1
	if withCount {
		n = count
		if n < 0 {
			n = 0
		}
		if n > len(members) {
			n = len(members)
		}
	}
	rand.Shuffle(len(members), func(i, j int) { members[i], members[j] = members[j], members[i] })
	popped := members[:n]
	_, _ = setRem(key, popped)
	return popped, nil
}

type setOpKind int

const (
	setOpInter setOpKind = iota
	setOpUnion
	setOpDiff
)

func setAsMap(obj *Obj) (map[string]struct{}, error) {
	if obj == nil {
		return map[string]struct{}{}, nil
	}
	if obj.Type != ObjSet {
		return nil, ErrWrongType
	}
	switch obj.Encoding {
	case EncodingIntset:
		return obj.Value.(*IntSet).toHashtable(), nil
	case EncodingHT:
		src := obj.Value.(map[string]struct{})
		out := make(map[string]struct{}, len(src))
		for k := range src {
			out[k] = struct{}{}
		}
		return out, nil
	default:
		return nil, ErrWrongType
	}
}

func setOp(kind setOpKind, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	baseObj := Get(keys[0])
	result, err := setAsMap(baseObj)
	if err != nil {
		return nil, err
	}

	for _, key := range keys[1:] {
		obj := Get(key)
		other, err := setAsMap(obj)
		if err != nil {
			return nil, err
		}
		switch kind {
		case setOpInter:
			for m := range result {
				if _, ok := other[m]; !ok {
					delete(result, m)
				}
			}
		case setOpUnion:
			for m := range other {
				result[m] = struct{}{}
			}
		case setOpDiff:
			for m := range other {
				delete(result, m)
			}
		}
	}

	out := make([]string, 0, len(result))
	for m := range result {
		out = append(out, m)
	}
	return out, nil
}

func setOpStore(kind setOpKind, dest string, keys []string) (int, error) {
	members, err := setOp(kind, keys)
	if err != nil {
		return 0, err
	}
	// Overwrite destination.
	if _, ok := store[dest]; ok {
		deleteKey(dest)
	}
	if len(members) == 0 {
		return 0, nil
	}
	if _, err := setAdd(dest, members); err != nil {
		return 0, err
	}
	n, err := setCard(dest)
	return n, err
}

// setMembersForAOF returns members for persistence.
func setMembersForAOF(obj *Obj) ([]string, error) {
	return setMembersObj(obj)
}
