package core

import (
	"errors"
	"io"
	"strconv"

	"github.com/sayandip18/redis-clone/internal/resp"
)

func evalAPPEND(args []string, w io.Writer) error {
	if len(args) != 2 {
		return errors.New("(error) ERR wrong number of arguments for 'append' command")
	}
	key, value := args[0], args[1]

	obj := Get(key)
	if obj == nil {
		obj = NewStringObj(value, -1)
		if err := stringObjToRaw(obj); err != nil {
			return err
		}
		if err := Put(key, obj); err != nil {
			return err
		}
		sds := obj.Value.(*SDS)
		return writeInt(w, int64(sds.Len()))
	}
	if obj.Type != ObjString {
		return ErrWrongType
	}
	if err := stringObjToRaw(obj); err != nil {
		return err
	}
	sds := obj.Value.(*SDS)
	sds.Cat(value)
	bumpVersion(key)
	return writeInt(w, int64(sds.Len()))
}

func evalSTRLEN(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'strlen' command")
	}
	obj := Get(args[0])
	if obj == nil {
		return writeInt(w, 0)
	}
	s, err := GetStringValue(obj)
	if err != nil {
		return err
	}
	return writeInt(w, int64(len(s)))
}

func evalGETRANGE(args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("(error) ERR wrong number of arguments for 'getrange' command")
	}
	start, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	end, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}

	obj := Get(args[0])
	if obj == nil {
		_, err := w.Write(resp.Encode("", false))
		return err
	}
	if obj.Type != ObjString {
		return ErrWrongType
	}

	var out string
	switch obj.Encoding {
	case EncodingRaw:
		sds, ok := obj.Value.(*SDS)
		if !ok || sds == nil {
			return ErrWrongType
		}
		out = sds.Range(int(start), int(end))
	default:
		s, err := GetStringValue(obj)
		if err != nil {
			return err
		}
		out = NewSDS(s).Range(int(start), int(end))
	}
	_, err = w.Write(resp.Encode(out, false))
	return err
}

func evalSETRANGE(args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("(error) ERR wrong number of arguments for 'setrange' command")
	}
	offset, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	if offset < 0 {
		return errors.New("(error) ERR offset is out of range")
	}

	key, value := args[0], args[2]
	obj := Get(key)
	if obj == nil {
		obj = &Obj{
			Type:      ObjString,
			Encoding:  EncodingRaw,
			Value:     NewSDS(""),
			ExpiresAt: -1,
			LRU:       getLRUClock(),
		}
		if err := Put(key, obj); err != nil {
			return err
		}
	} else {
		if obj.Type != ObjString {
			return ErrWrongType
		}
		if err := stringObjToRaw(obj); err != nil {
			return err
		}
		bumpVersion(key)
	}

	sds := obj.Value.(*SDS)
	n := sds.SetRange(int(offset), value)
	return writeInt(w, int64(n))
}
