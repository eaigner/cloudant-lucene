package cloudant

import (
  "bytes"
	"reflect"
	"strconv"
	"strings"
)

const (
	Or        Keyword = "OR"
	And       Keyword = "AND"
	Infinity  Keyword = "Infinity"
	NInfinity Keyword = "-Infinity"
)

type Query struct {
	isGroup bool
	c       queryComponents
}

// ParseQuery parses a string into a Query. If the query string is invalid, it
// returns a initialized empty query and false. The index map is used to map
// query to real index names.
func ParseQuery(query string, indexMap map[string]string) (*Query, bool) {
	c := strings.Split(query, " ")
	q := new(Query)

	i := 0
	for _, part := range c {
		if len(part) > 0 {
			c2 := strings.SplitN(part, ":", 2)
			if len(c2) == 2 {
				for k, v := range indexMap {
					if k == c2[0] {
						if v != "" {
							k = v
						}
						if i > 0 {
							q.And()
						}
						q.Index(k).Is(c2[1])
						i++
						break
					}
				}
			} else {
				if i > 0 {
					q.And()
				}
				q.Index("").Is(part)
				i++
			}
		}
	}

	return q, (i > 0)
}

func (q *Query) Group() *Query {
	n := new(Query)
	n.isGroup = true
	q.c = append(q.c, n)
	return n
}

func (q *Query) Index(name string) *Index {
	n := new(Index)
	n.name = name
	q.c = append(q.c, n)
	return n
}

func (q *Query) And() {
	q.c = append(q.c, And)
}

func (q *Query) Or() {
	q.c = append(q.c, Or)
}

func (q *Query) String() string {
	if q.isGroup {
		return `(` + q.c.String() + `)`
	}
	return q.c.String()
}

type Index struct {
	name string
	c    queryComponents
}

func (i *Index) And() *Index {
	i.c = append(i.c, And)
	return i
}

func (i *Index) Or() *Index {
	i.c = append(i.c, Or)
	return i
}

func (i *Index) Is(v interface{}) *Index {
	i.c = append(i.c, queryValue{v})
	return i
}

func (i *Index) Range(a, b interface{}) *Index {
	i.c = append(i.c, queryRange{
		[2]queryComponent{
			queryValue{a},
			queryValue{b},
		},
	})
	return i
}

func (i *Index) String() string {
	n := i.name
	if n == "" {
		n = "default"
	}
	return n + `:(` + i.c.String() + `)`
}

type queryComponent interface {
	String() string
}

type queryComponents []queryComponent

func (c queryComponents) String() string {
	a := make([]string, 0, len(c))
	for _, v := range c {
		a = append(a, v.String())
	}
	return strings.Join(a, " ")
}

// queryValue represents a user value
type queryValue struct {
	t interface{}
}

func (v queryValue) String() string {
	switch t := v.t.(type) {
	case Keyword:
		return t.String()
	case string:
		return `"` + Escape(t) + `"`
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(t).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(t).Uint(), 10)
	default:
		panic("invalid query value type")
	}
}

type queryRange struct {
	v [2]queryComponent
}

func (r queryRange) String() string {
	return `[` + r.v[0].String() + ` TO ` + r.v[1].String() + `]`
}

// Keyword represents a keyword that must not be escaped
type Keyword string

func (w Keyword) String() string {
	return string(w)
}

// Escape escapes a string for a lucene query
func Escape(s string) string {
	var b bytes.Buffer
	b.Grow(len(s) * 2)
	for _, r := range s {
		switch r {
		// Although lucene doesn't require '/' to be escaped, the cloudant docs do
		case '\\', '/', '+', '-', '!', '(', ')', ':', '^', '[', ']', '"', '{', '}', '~', '*', '?', '|', '&':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
