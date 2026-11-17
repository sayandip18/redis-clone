package core

import (
	"errors"
	"io"
	"strconv"
)

func evalLPUSH(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'lpush' command")
	}
	n, err := listPush(args[0], listLeft, args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalRPUSH(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'rpush' command")
	}
	n, err := listPush(args[0], listRight, args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalLPOP(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'lpop' command")
	}
	v, ok, err := listPop(args[0], listLeft)
	if err != nil {
		return err
	}
	return writeBulkOrNil(w, v, ok)
}

func evalRPOP(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'rpop' command")
	}
	v, ok, err := listPop(args[0], listRight)
	if err != nil {
		return err
	}
	return writeBulkOrNil(w, v, ok)
}

func evalLLEN(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'llen' command")
	}
	n, err := listLen(args[0])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalLRANGE(args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("(error) ERR wrong number of arguments for 'lrange' command")
	}
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	vals, err := listRange(args[0], start, stop)
	if err != nil {
		return err
	}
	if vals == nil {
		vals = []string{}
	}
	return writeBulkArray(w, vals)
}

func evalLINDEX(args []string, w io.Writer) error {
	if len(args) != 2 {
		return errors.New("(error) ERR wrong number of arguments for 'lindex' command")
	}
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	v, ok, err := listIndex(args[0], idx)
	if err != nil {
		return err
	}
	return writeBulkOrNil(w, v, ok)
}

func evalLSET(args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("(error) ERR wrong number of arguments for 'lset' command")
	}
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	if err := listSet(args[0], idx, args[2]); err != nil {
		return err
	}
	return writeOK(w)
}

func evalLINSERT(args []string, w io.Writer) error {
	if len(args) != 4 {
		return errors.New("(error) ERR wrong number of arguments for 'linsert' command")
	}
	var before bool
	switch args[1] {
	case "BEFORE", "before":
		before = true
	case "AFTER", "after":
		before = false
	default:
		return errors.New("(error) ERR syntax error")
	}
	n, err := listInsert(args[0], before, args[2], args[3])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalLTRIM(args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("(error) ERR wrong number of arguments for 'ltrim' command")
	}
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	if err := listTrim(args[0], start, stop); err != nil {
		return err
	}
	return writeOK(w)
}

func evalLREM(args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("(error) ERR wrong number of arguments for 'lrem' command")
	}
	count, err := strconv.Atoi(args[1])
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	n, err := listRem(args[0], count, args[2])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}
