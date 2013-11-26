package ccd

import (
	"fmt"
	"github.com/jteeuwen/go-pkg-xmlx"
	"menteslibres.net/gosexy/to"
	"strings"
	"time"
)

const (
	// Found both these formats in the wild
	TimeDecidingIndex = 14
	TimeFormat        = "20060102150405-0700"
	TimeFormat2       = "20060102150405.000-0700"
)

type TimeType string

const (
	// represents a single point in time
	TIME_SINGLE TimeType = "TS"
	// interval of time
	TIME_INTERVAL = "IVL_TS"
	// periodic interval of time
	TIME_PERIODIC = "PIVL_TS"
	// event based time interval
	TIME_EVENT = "EIVL_TS"
	// represents an probabilistic time interval and is used to represent dosing frequencies like q4-6h
	TIME_PROBABILISTIC = "PIVL_PPD_TS"
	// represents a parenthetical set of time expressions
	TIME_PARENTHETICAL = "SXPR_TS"
)

type Time struct {
	Type   TimeType
	Low    time.Time
	High   time.Time
	Value  time.Time
	Period time.Duration // s, min, h, d, wk and mo
}

func (t *Time) IsZero() bool {
	return t.Value.IsZero() && t.Low.IsZero() && t.High.IsZero() && t.Period == 0
}

func ParseTimeNode(node *xmlx.Node) (t Time) {
	if node == nil {
		return t
	}

	t.Type = TimeType(strings.ToUpper(node.As("*", "type")))

	lowNode := Nget(node, "low")
	if lowNode != nil {
		t.Low, _ = ParseHL7Time(lowNode.As("*", "value"))
	}
	highNode := Nget(node, "high")
	if highNode != nil {
		t.High, _ = ParseHL7Time(highNode.As("*", "value"))
	}

	val := node.As("*", "value")
	if len(val) > 0 {
		t.Value, _ = ParseHL7Time(val)
	} else {
		centerNode := Nget(node, "center")
		if centerNode != nil {
			t.Value, _ = ParseHL7Time(centerNode.As("*", "value"))
		}
	}

	if t.Value.IsZero() && !t.Low.IsZero() && t.High.IsZero() {
		t.Value = t.Low
	}

	period := Nget(node, "period")
	if period != nil {
		value := time.Duration(to.Int64(period.As("*", "value")))
		unit := period.As("*", "unit")
		switch strings.ToLower(unit) {
		case "s":
			t.Period = time.Second * value
		case "min":
			t.Period = time.Minute * value
		case "h":
			t.Period = time.Hour * value
		case "d":
			t.Period = time.Hour * 24 * value
		case "wk":
			t.Period = time.Hour * 24 * 7 * value
		case "mo":
			t.Period = time.Hour * 24 * 30 * value
		}
	}

	return t
}

// Dates and times in a CCD can be partial. Meaning they can be:
//   2006, 200601, 20060102, etc...
// This function helps us parse all cases.
func ParseHL7Time(value string) (time.Time, error) {
	l := len(value)
	tmfmt := TimeFormat
	if l > TimeDecidingIndex && value[TimeDecidingIndex] == '.' {
		tmfmt = TimeFormat2
	}
	return time.Parse(tmfmt[:l], value)
}

// Node get.
// helper function to continually transverse down the
// xml nodes in args, and return the last one.
func Nget(node *xmlx.Node, args ...string) *xmlx.Node {
	for _, a := range args {
		if node == nil {
			return nil
		}

		node = node.SelectNode("*", a)
	}

	return node
}

// Node Safe get.
// just like Nget, but returns a node no matter what.
func Nsget(node *xmlx.Node, args ...string) *xmlx.Node {
	n := Nget(node, args...)
	if n == nil {
		return xmlx.NewNode(0)
	}
	return n
}

func insertSortParser(p Parser, parsers Parsers) Parsers {
	i := len(parsers) - 1
	for ; i >= 0; i-- {
		if p.Priority > parsers[i].Priority {
			i += 1
			break
		}
	}

	if i < 0 {
		i = 0
	}

	parsers = append(parsers, p) // this just expands storage.
	copy(parsers[i+1:], parsers[i:])
	parsers[i] = p

	return parsers
}

func templateId(node *xmlx.Node) string {
	idNodes := node.SelectNodes("*", "templateId")
	id := ""
	for _, idNode := range idNodes {
		id = idNode.As("*", "root")
		if strings.HasPrefix(id, "2.16.840.1.113883.10.20.") {
			return id
		}
	}

	return id
}

func codeSystemToMedType(codeSystem string) (string, error) {
	switch codeSystem {
	case "2.16.840.1.113883.6.69": // NDC
		return "NDC", nil
	case "2.16.840.1.113883.6.88": // RxNorm
		return "RxNorm", nil
	}
	return "", fmt.Errorf(`Unknown med codeSystem value of "%s"`, codeSystem)
}