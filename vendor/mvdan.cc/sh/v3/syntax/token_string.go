// Code generated by "stringer -type token -linecomment -trimprefix _"; DO NOT EDIT.

package syntax

import "strconv"

const _token_name = "illegalTokEOFNewlLitLitWordLitRedir'\"`&&&||||&$$'$\"${$[$($(([[[(((}])));;;;&;;&;|!++--***==!=<=>=+=-=*=/=%=&=|=^=<<=>>=>>><<><&>&>|<<<<-<<<&>&>><(>(+:+-:-?:?=:=%%%###^^^,,,@///:-e-f-d-c-b-p-S-L-k-g-u-G-O-N-r-w-x-s-t-z-n-o-v-R=~-nt-ot-ef-eq-ne-le-ge-lt-gt?(*(+(@(!("

var _token_index = [...]uint16{0, 10, 13, 17, 20, 27, 35, 36, 37, 38, 39, 41, 43, 44, 46, 47, 49, 51, 53, 55, 57, 60, 61, 63, 64, 66, 67, 68, 69, 71, 72, 74, 76, 79, 81, 82, 84, 86, 87, 89, 91, 93, 95, 97, 99, 101, 103, 105, 107, 109, 111, 113, 116, 119, 120, 122, 123, 125, 127, 129, 131, 133, 136, 139, 141, 144, 146, 148, 149, 151, 152, 154, 155, 157, 158, 160, 161, 163, 164, 166, 167, 169, 170, 172, 173, 174, 176, 177, 179, 181, 183, 185, 187, 189, 191, 193, 195, 197, 199, 201, 203, 205, 207, 209, 211, 213, 215, 217, 219, 221, 223, 225, 227, 230, 233, 236, 239, 242, 245, 248, 251, 254, 256, 258, 260, 262, 264}

func (i token) String() string {
	if i >= token(len(_token_index)-1) {
		return "token(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _token_name[_token_index[i]:_token_index[i+1]]
}
