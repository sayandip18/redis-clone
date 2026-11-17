package core

import (
	"errors"
	"io"
	"strconv"
)

func evalSADD(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'sadd' command")
	}
	n, err := setAdd(args[0], args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalSREM(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'srem' command")
	}
	n, err := setRem(args[0], args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalSISMEMBER(args []string, w io.Writer) error {
	if len(args) != 2 {
		return errors.New("(error) ERR wrong number of arguments for 'sismember' command")
	}
	ok, err := setIsMember(args[0], args[1])
	if err != nil {
		return err
	}
	if ok {
		return writeInt(w, 1)
	}
	return writeInt(w, 0)
}

func evalSCARD(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'scard' command")
	}
	n, err := setCard(args[0])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalSMEMBERS(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'smembers' command")
	}
	members, err := setMembers(args[0])
	if err != nil {
		return err
	}
	if members == nil {
		members = []string{}
	}
	return writeBulkArray(w, members)
}

func evalSRANDMEMBER(args []string, w io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New("(error) ERR wrong number of arguments for 'srandmember' command")
	}
	withCount := len(args) == 2
	count := 1
	if withCount {
		c, err := strconv.Atoi(args[1])
		if err != nil {
			return errors.New("(error) ERR value is not an integer or out of range")
		}
		count = c
	}
	members, err := setRandom(args[0], count, withCount)
	if err != nil {
		return err
	}
	if !withCount {
		if members == nil {
			_, err := w.Write(RESP_NIL)
			return err
		}
		return writeBulkOrNil(w, members[0], true)
	}
	return writeBulkArray(w, members)
}

func evalSPOP(args []string, w io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New("(error) ERR wrong number of arguments for 'spop' command")
	}
	withCount := len(args) == 2
	count := 1
	if withCount {
		c, err := strconv.Atoi(args[1])
		if err != nil {
			return errors.New("(error) ERR value is not an integer or out of range")
		}
		if c < 0 {
			return errors.New("(error) ERR value is out of range")
		}
		count = c
	}
	members, err := setPop(args[0], count, withCount)
	if err != nil {
		return err
	}
	if !withCount {
		if members == nil {
			_, err := w.Write(RESP_NIL)
			return err
		}
		return writeBulkOrNil(w, members[0], true)
	}
	return writeBulkArray(w, members)
}

func evalSINTER(args []string, w io.Writer) error {
	if len(args) < 1 {
		return errors.New("(error) ERR wrong number of arguments for 'sinter' command")
	}
	members, err := setOp(setOpInter, args)
	if err != nil {
		return err
	}
	return writeBulkArray(w, members)
}

func evalSUNION(args []string, w io.Writer) error {
	if len(args) < 1 {
		return errors.New("(error) ERR wrong number of arguments for 'sunion' command")
	}
	members, err := setOp(setOpUnion, args)
	if err != nil {
		return err
	}
	return writeBulkArray(w, members)
}

func evalSDIFF(args []string, w io.Writer) error {
	if len(args) < 1 {
		return errors.New("(error) ERR wrong number of arguments for 'sdiff' command")
	}
	members, err := setOp(setOpDiff, args)
	if err != nil {
		return err
	}
	return writeBulkArray(w, members)
}

func evalSINTERSTORE(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'sinterstore' command")
	}
	n, err := setOpStore(setOpInter, args[0], args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalSUNIONSTORE(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'sunionstore' command")
	}
	n, err := setOpStore(setOpUnion, args[0], args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}

func evalSDIFFSTORE(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New("(error) ERR wrong number of arguments for 'sdiffstore' command")
	}
	n, err := setOpStore(setOpDiff, args[0], args[1:])
	if err != nil {
		return err
	}
	return writeInt(w, int64(n))
}
