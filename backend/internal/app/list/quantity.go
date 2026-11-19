package list

import (
	"encoding/json"
	"regexp"
	"strings"
)

type Unit string

const (
	UnitPiece      Unit = "st"
	UnitPackage    Unit = "pkt"
	UnitKilogram   Unit = "kg"
	UnitGram       Unit = "g"
	UnitLiter      Unit = "l"
	UnitDeciliter  Unit = "dl"
	UnitCentiliter Unit = "cl"
	UnitMilliliter Unit = "ml"
	UnitTablespoon Unit = "msk"
	UnitTeaspoon   Unit = "tsk"
	UnitKryddmått  Unit = "krm"
)

func (unit Unit) Valid() bool {
	switch unit {
	case UnitPiece,
		UnitPackage,
		UnitKilogram,
		UnitGram,
		UnitLiter,
		UnitDeciliter,
		UnitCentiliter,
		UnitMilliliter,
		UnitTablespoon,
		UnitTeaspoon,
		UnitKryddmått:
		return true
	default:
		return false
	}
}

const MaxAmountLength = 100

var decimalAmount = regexp.MustCompile(`^[0-9]+([.][0-9]+)?$`)

// Keep decimals exact instead of rounding them through a floating-point value.
func NormalizeAmount(amount *string) bool {
	if amount == nil {
		return true
	}

	*amount = strings.ReplaceAll(strings.TrimSpace(*amount), ",", ".")

	return len(*amount) <= MaxAmountLength && decimalAmount.MatchString(*amount) &&
		strings.Trim(*amount, "0.") != ""
}

// Unlike optional, a nullable field distinguishes omission from an explicit clear.
type nullable[T any] struct {
	set   bool
	value *T
}

func (field *nullable[T]) UnmarshalJSON(data []byte) error {
	field.set = true
	return json.Unmarshal(data, &field.value)
}
