package resp

import (
	"errors"
	"fmt"
)

var (
	ErrIncomplete = errors.New("incomplete RESP data")
	ErrInvalid    = errors.New("invalid RESP data")
)

func DecodeArrayString(data []byte) ([]string, error) {
	value, err := Decode(data)
	if err != nil {
		return nil, err
	}

	tokens, err := arrayToStrings(value)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// DrainArrayStrings parses consecutive RESP arrays of strings from data.
// Incomplete trailing bytes are returned in rest with a nil error.
func DrainArrayStrings(data []byte) (commands [][]string, rest []byte, err error) {
	rest = data
	for len(rest) > 0 {
		value, n, err := DecodeOne(rest)
		if err != nil {
			if errors.Is(err, ErrIncomplete) {
				return commands, rest, nil
			}
			return nil, rest, err
		}

		tokens, err := arrayToStrings(value)
		if err != nil {
			return nil, rest, err
		}
		commands = append(commands, tokens)
		rest = rest[n:]
	}
	return commands, rest, nil
}

func arrayToStrings(value interface{}) ([]string, error) {
	ts, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: expected array", ErrInvalid)
	}

	tokens := make([]string, len(ts))
	for i := range ts {
		s, ok := ts[i].(string)
		if !ok {
			return nil, fmt.Errorf("%w: expected string element", ErrInvalid)
		}
		tokens[i] = s
	}
	return tokens, nil
}

// Decode parses a complete RESP value from data.
func Decode(data []byte) (interface{}, error) {
	value, _, err := DecodeOne(data)
	return value, err
}

// DecodeOne parses the first RESP value in data and returns how many bytes were consumed.
func DecodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, ErrIncomplete
	}

	switch data[0] {
	case '+':
		return decodeSimpleString(data)
	case '-':
		return decodeError(data)
	case ':':
		return decodeInteger(data)
	case '$':
		return decodeBulkString(data)
	case '*':
		return decodeArray(data)
	default:
		return nil, 0, fmt.Errorf("%w: unknown type byte %q", ErrInvalid, data[0])
	}
}

func decodeSimpleString(data []byte) (interface{}, int, error) {
	line, n, err := readLine(data)
	if err != nil {
		return nil, 0, err
	}
	return string(line[1:]), n, nil
}

func decodeError(data []byte) (interface{}, int, error) {
	line, n, err := readLine(data)
	if err != nil {
		return nil, 0, err
	}
	return string(line[1:]), n, nil
}

func decodeInteger(data []byte) (interface{}, int, error) {
	line, n, err := readLine(data)
	if err != nil {
		return nil, 0, err
	}

	payload := line[1:]
	if len(payload) == 0 {
		return nil, 0, fmt.Errorf("%w: empty integer", ErrInvalid)
	}

	var value int64
	sign := int64(1)
	i := 0
	if payload[0] == '-' {
		sign = -1
		i = 1
		if len(payload) == 1 {
			return nil, 0, fmt.Errorf("%w: empty integer", ErrInvalid)
		}
	} else if payload[0] == '+' {
		i = 1
		if len(payload) == 1 {
			return nil, 0, fmt.Errorf("%w: empty integer", ErrInvalid)
		}
	}

	for ; i < len(payload); i++ {
		c := payload[i]
		if c < '0' || c > '9' {
			return nil, 0, fmt.Errorf("%w: non-digit in integer", ErrInvalid)
		}
		digit := int64(c - '0')
		if value > (1<<63-1-digit)/10 {
			return nil, 0, fmt.Errorf("%w: integer overflow", ErrInvalid)
		}
		value = value*10 + digit
	}

	return sign * value, n, nil
}

func decodeBulkString(data []byte) (interface{}, int, error) {
	line, n, err := readLine(data)
	if err != nil {
		return nil, 0, err
	}

	length, err := parseLength(line[1:])
	if err != nil {
		return nil, 0, err
	}
	if length == -1 {
		return nil, n, nil
	}

	end := n + length + 2
	if len(data) < end {
		return nil, 0, ErrIncomplete
	}
	if data[n+length] != '\r' || data[n+length+1] != '\n' {
		return nil, 0, fmt.Errorf("%w: bulk string missing CRLF", ErrInvalid)
	}

	return string(data[n : n+length]), end, nil
}

func decodeArray(data []byte) (interface{}, int, error) {
	line, n, err := readLine(data)
	if err != nil {
		return nil, 0, err
	}

	count, err := parseLength(line[1:])
	if err != nil {
		return nil, 0, err
	}
	if count == -1 {
		return nil, n, nil
	}

	items := make([]interface{}, 0, count)
	pos := n
	for i := 0; i < count; i++ {
		value, consumed, err := DecodeOne(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		items = append(items, value)
		pos += consumed
	}

	return items, pos, nil
}

// readLine returns the line without the trailing CRLF, and the number of bytes
// consumed including CRLF.
func readLine(data []byte) ([]byte, int, error) {
	for i := 0; i < len(data); i++ {
		if data[i] != '\r' {
			continue
		}
		if i+1 >= len(data) {
			return nil, 0, ErrIncomplete
		}
		if data[i+1] != '\n' {
			return nil, 0, fmt.Errorf("%w: expected LF after CR", ErrInvalid)
		}
		if i == 0 {
			return nil, 0, fmt.Errorf("%w: empty line", ErrInvalid)
		}
		return data[:i], i + 2, nil
	}
	return nil, 0, ErrIncomplete
}

func parseLength(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, fmt.Errorf("%w: empty length", ErrInvalid)
	}

	if payload[0] == '-' {
		if len(payload) == 2 && payload[1] == '1' {
			return -1, nil
		}
		return 0, fmt.Errorf("%w: invalid negative length", ErrInvalid)
	}

	var length int
	for _, c := range payload {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("%w: non-digit in length", ErrInvalid)
		}
		digit := int(c - '0')
		if length > (int(^uint(0)>>1)-digit)/10 {
			return 0, fmt.Errorf("%w: length overflow", ErrInvalid)
		}
		length = length*10 + digit
	}
	return length, nil
}
