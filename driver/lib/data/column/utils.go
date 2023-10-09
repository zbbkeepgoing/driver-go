package column

import (
	"encoding/binary"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	emptyString        = ""
	escape             = '\\'
	roundOpenBracket   = '('
	roundCloseBracket  = ')'
	squareOpenBracket  = '['
	squareCloseBracket = ']'
	curlyOpenBracket   = '{'
	curlyCloseBracket  = '}'
	singleQuote        = '\''
	doubleQuote        = '"'
	backQuote          = '`'
	comma              = ','
	space              = ' '
	colon              = ':'

	listSeparator = ", "
	mapSeparator  = ":"
	enumSeparator = "="

	localtime = "localtime"
	local     = "Local"
)

// commaIterator takes in a string to create a string iterator which separates whole string by comma
// For each time it's called, returns string and bool.
// If end is reached, returns empty string and false
// Note: takes into account square brackets, round bracket, escape character, single quote, back quote and double quote
// eg.
// 	s := "[a,d,f,f], dfsdfsd,fsdfsdf,sdfd"
//	iter := parseUntilCommaOrEOF2(s)
//	for {
//		s, ok := iter()
//		if !ok {
//			break
//		}
//		fmt.Println(s)
//	}
//  // output:
//  // [a,d,f,f]
//  // dfsdfsd
//  // fsdfsdf
//  // sdfd
func commaIterator(s string) func() (string, bool) {
	var (
		escaped                                                  bool
		quoteType                                                byte
		roundBracketCount, squareBracketCount, curlyBracketCount int
		currentIdx                                               int
	)

	inAnyBracket := func() bool {
		return !(roundBracketCount == 0 && squareBracketCount == 0 && curlyBracketCount == 0)
	}
	inAnyQuote := func() bool {
		return quoteType != 0
	}

	var lastIdx int

	return func() (string, bool) {
		for currentIdx < len(s) {
			if escaped {
				escaped = false
				goto nextPosition
			}
			if s[currentIdx] == escape {
				escaped = true
				goto nextPosition
			}

			if inAnyQuote() {
				if s[currentIdx] == quoteType {
					quoteType = 0
				}
				goto nextPosition
			}

			switch s[currentIdx] {
			case squareOpenBracket:
				squareBracketCount++
			case squareCloseBracket:
				squareBracketCount--
			case roundOpenBracket:
				roundBracketCount++
			case roundCloseBracket:
				roundBracketCount--
			case curlyOpenBracket:
				curlyBracketCount++
			case curlyCloseBracket:
				curlyBracketCount--
			case singleQuote:
				quoteType = singleQuote
			case doubleQuote:
				quoteType = doubleQuote
			case backQuote:
				quoteType = backQuote
			case comma:
				if inAnyBracket() {
					goto nextPosition
				}
				result := s[lastIdx:currentIdx]
				currentIdx++
				lastIdx = currentIdx
				return strings.TrimSpace(result), true
			}
		nextPosition:
			currentIdx++
		}

		if currentIdx == len(s) { // return last index when current index reaches the end
			result := s[lastIdx:currentIdx]
			currentIdx++ // any more of this function will return 0 instead
			return strings.TrimSpace(result), true
		}
		return emptyString, false
	}
}

// splitIgnoreBraces splits string separated by separator, accounting for braces, escape and quote.
// Separator should be a char
func splitIgnoreBraces(src string, separator byte, bufferReuse []string) []string {
	bufferReuse = bufferReuse[:0]

	var (
		currentIdx int
		lastIdx    int
	)
	for currentIdx < len(src) { // special handling for last idx
		switch c := src[currentIdx]; c {
		case escape:
			currentIdx++
		case squareOpenBracket:
			currentIdx += indexOfMatchCloseBracket(src[currentIdx:], squareOpenBracket, squareCloseBracket)
		case roundOpenBracket:
			currentIdx += indexOfMatchCloseBracket(src[currentIdx:], roundOpenBracket, roundCloseBracket)
		case curlyOpenBracket:
			currentIdx += indexOfMatchCloseBracket(src[currentIdx:], curlyOpenBracket, curlyCloseBracket)
		case singleQuote, doubleQuote, backQuote:
			currentIdx += 1 + indexTillByteOrEOF(src[currentIdx+1:], c)
		case separator:
			result := src[lastIdx:currentIdx]
			lastIdx = currentIdx + 1
			bufferReuse = append(bufferReuse, strings.TrimSpace(result))
		}
		currentIdx++
	}

	bufferReuse = append(bufferReuse, strings.TrimSpace(src[lastIdx:]))

	return bufferReuse
}

// Given s[0] is equal to openBracket,
// indexOfMatchCloseBracket find the position i that s[i] == closeBracket that if we only keeps those brackets from s[0:i+1],
// that string will be balanced bracket
// if can't find any index, return len(s) - 1;
func indexOfMatchCloseBracket(s string, openBracket byte, closeBracket byte) int {
	openCount := 0
	isEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == escape {
			isEscape = true
			continue
		}

		if s[i] == openBracket && !isEscape {
			openCount++
			isEscape = false
			continue
		}
		if s[i] == closeBracket && !isEscape {
			openCount--
		}

		if openCount == 0 {
			return i
		}

		isEscape = false
	}

	return len(s) - 1
}

// indexTillByteOrEOF return the position where next byte occur or EOF.
// Checks that given byte is not escaped
func indexTillByteOrEOF(s string, c byte) int {
	var i int
	for {
		i += strings.IndexByte(s[i:], c)
		switch i {
		case 0, len(s) - 1:
			return i
		case -1:
			return len(s)
		}
		if s[i-1] != escape {
			return i
		}
		i++
	}
}

// indexTillNotByteOrEOF returns the first index where the byte is not equals to c
func indexTillNotByteOrEOF(s string, c byte) int {
	var i int
	// While i < size of s
	for i = 0; i < len(s); i++ {
		// If i not equals to byte return the index
		if s[i] != c {
			return i
		}
	}

	// If not return last index
	return len(s) - 1
}

func bufferRowToUint16(b []byte, row int) uint16 {
	b = b[row*2 : (row+1)*2]
	return binary.LittleEndian.Uint16(b)
}

func bufferRowToUint32(b []byte, row int) uint32 {
	b = b[row*4 : (row+1)*4]
	return binary.LittleEndian.Uint32(b)
}

func bufferRowToUint64(b []byte, row int) uint64 {
	b = b[row*8 : (row+1)*8]
	return binary.LittleEndian.Uint64(b)
}

func getRowRaw(raw []byte, row int, rowSize int) []byte {
	return raw[row*rowSize : (row+1)*rowSize]
}

func getDateTimeLocation(t CHColumnType) (*time.Location, error) {
	if len(t) < 9 { // DateTime
		return nil, nil
	}
	tzString := string(t[10 : len(t)-2]) // DateTime('Europe/Moscow')
	if tzString == localtime {
		tzString = local
	}
	location, err := time.LoadLocation(tzString)
	if err != nil {
		return nil, err
	}
	return location, nil
}

func parseDateTime64Param(t CHColumnType) (int, string, error) {
	params := strings.Split(string(t[11:len(t)-1]), ",") // DateTime64(23, timestamp), e.g. DateTime64(3, 'Europe/Moscow')
	precisionString := strings.TrimSpace(params[0])
	if len(precisionString) == 0 {
		return 0, "", nil
	}
	precision, err := strconv.ParseUint(precisionString, 10, 32)
	if err != nil {
		return 0, "", err
	}
	if len(params) == 1 {
		return int(precision), "", nil
	}
	tzString := strings.Trim(strings.TrimSpace(params[1]), "'")
	return int(precision), tzString, nil
}

func getDateTime64Param(t CHColumnType) (int, *time.Location, error) {

	precision, tzString, err := parseDateTime64Param(t)
	if err != nil {
		return 0, nil, err
	}
	if tzString == "" {
		return precision, nil, nil
	}

	if tzString == localtime {
		tzString = local
	}
	tz, err := time.LoadLocation(tzString)
	if err != nil {
		return 0, nil, err
	}
	return precision, tz, nil
}

func getTimeParam(t CHColumnType) (int, error) {

	scaleString := string(t[5 : len(t)-1]) //Time(3)
	if len(scaleString) == 0 {
		return 0, nil
	}
	scale, err := strconv.ParseInt(scaleString, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(scale), nil
}

func getColumnValuesUsingOffset(start, end int, columnData CHColumnData) []interface{} {
	result := make([]interface{}, end-start)
	for i := start; i < end; i++ {
		result[i-start] = columnData.GetValue(i)
	}
	return result
}

func getColumnStringsUsingOffset(start, end int, columnData CHColumnData) []string {
	result := make([]string, end-start)
	for i := start; i < end; i++ {
		result[i-start] = columnData.GetString(i)
	}
	return result
}

// processString optionally removes the string wrapped in quotes
func processString(s string) string {
	if len(s) < 2 {
		return s
	}
	quote := s[0]
	switch quote {
	case singleQuote, doubleQuote, backQuote:
	default:
		return s
	}
	if quote != s[len(s)-1] {
		return s
	}
	return s[1 : len(s)-1]
}

func builderWriteKind(builder *strings.Builder, value string, valueType reflect.Kind) {
	if valueType != reflect.String {
		builder.WriteString(value)
		return
	}

	builder.WriteByte(singleQuote)
	builder.WriteString(value)
	builder.WriteByte(singleQuote)
}
