package httpapi

import (
	"errors"
	"net/url"
	"strconv"
)

var errInvalidQuery = errors.New("invalid query parameters")

type listQuery struct {
	limit  int
	offset int
	status string
}

func parseListQuery(rawQuery string) (listQuery, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return listQuery{}, errInvalidQuery
	}

	result := listQuery{limit: 20}

	for name, values := range query {
		if len(values) != 1 {
			return listQuery{}, errInvalidQuery
		}

		switch name {
		case "status":
			if values[0] != "new" && values[0] != "cancelled" {
				return listQuery{}, errInvalidQuery
			}
			result.status = values[0]

		case "limit", "offset":
			value, err := strconv.ParseUint(values[0], 10, strconv.IntSize-1)
			if err != nil {
				return listQuery{}, errInvalidQuery
			}

			if name == "limit" {
				if value < 1 || value > 100 {
					return listQuery{}, errInvalidQuery
				}
				result.limit = int(value)
			} else {
				result.offset = int(value)
			}

		default:
			return listQuery{}, errInvalidQuery
		}
	}

	return result, nil
}
