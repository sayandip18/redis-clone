package resp

import "fmt"

var RESP_NIL = []byte("$-1\r\n")

func Encode(value interface{}, isSimple bool) []byte {
	switch v := value.(type) {
		case string:
			if isSimple {
				return []byte(fmt.Sprintf("+%s\r\n", v))
			}
			return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v), v))
		case int64:
			return []byte(fmt.Sprintf(":%d\r\n", v))
		default:
			return RESP_NIL
	}
}
