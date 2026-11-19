package list

import (
	"context"
	"math/big"
	"strings"

	"github.com/herrnan/handlat/internal/app/list/db"
)

func addOrMergeItem(ctx context.Context, q *db.Queries, item db.AddItemParams) (string, error) {
	if item.Amount == nil {
		return "", q.AddItem(ctx, item)
	}

	candidates, err := q.ListMergeCandidates(ctx, db.ListMergeCandidatesParams{
		ListID:     item.ListID,
		IncomingID: item.ID,
	})
	if err != nil {
		return "", err
	}

	var mergedItemID string

	name := strings.Join(strings.Fields(item.Text), " ")
	for _, candidate := range candidates {
		if candidate.Amount == nil ||
			!strings.EqualFold(name, strings.Join(strings.Fields(candidate.Text), " ")) {
			continue
		}

		amount, unit, ok := mergeAmounts(
			*candidate.Amount, *item.Amount, unitValue(candidate.Unit), unitValue(item.Unit),
		)
		if !ok {
			continue
		}

		var savedUnit *string
		if unit != "" {
			savedUnit = new(string(unit))
		}

		if err := q.UpdateItem(ctx, db.UpdateItemParams{
			ListID:    item.ListID,
			ID:        candidate.ID,
			SetAmount: 1,
			Amount:    &amount,
			SetUnit:   1,
			Unit:      savedUnit,
		}); err != nil {
			return "", err
		}

		// Retain the incoming ID so that a retry cannot add the amount again.
		item.DeletedAt = &item.CreatedAt
		mergedItemID = candidate.ID

		break
	}

	return mergedItemID, q.AddItem(ctx, item)
}

func unitValue(unit *string) Unit {
	if unit == nil {
		return ""
	}

	return Unit(*unit)
}

func mergeAmounts(target, incoming string, targetUnit, incomingUnit Unit) (string, Unit, bool) {
	var base Unit

	targetFactor, incomingFactor := int64(1), int64(1)

	if targetUnit != incomingUnit {
		var incomingBase Unit

		base, targetFactor = metricScale(targetUnit)
		incomingBase, incomingFactor = metricScale(incomingUnit)

		if base == "" || base != incomingBase {
			return "", "", false
		}
	}

	sum, ok := new(big.Rat).SetString(target)
	if !ok {
		return "", "", false
	}

	add, ok := new(big.Rat).SetString(incoming)
	if !ok {
		return "", "", false
	}

	sum.Add(sum, add.Mul(add, big.NewRat(incomingFactor, targetFactor)))

	if amount, ok := exactDecimal(sum); ok {
		return amount, targetUnit, true
	}

	if base != "" {
		amount, ok := exactDecimal(sum.Mul(sum, big.NewRat(targetFactor, 1)))

		return amount, base, ok
	}

	return "", "", false
}

func exactDecimal(value *big.Rat) (string, bool) {
	digits, exact := value.FloatPrec()
	if !exact {
		return "", false
	}

	amount := value.FloatString(digits)

	return amount, len(amount) <= maxAmountLength
}

func metricScale(unit Unit) (Unit, int64) {
	switch unit {
	case UnitKilogram:
		return UnitGram, 1000
	case UnitGram:
		return UnitGram, 1
	case UnitLiter:
		return UnitMilliliter, 1000
	case UnitDeciliter:
		return UnitMilliliter, 100
	case UnitCentiliter:
		return UnitMilliliter, 10
	case UnitMilliliter, UnitKryddmått:
		return UnitMilliliter, 1
	case UnitTablespoon:
		return UnitMilliliter, 15
	case UnitTeaspoon:
		return UnitMilliliter, 5
	default:
		return "", 0
	}
}
