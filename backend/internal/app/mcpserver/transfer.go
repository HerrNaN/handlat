package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func transferIngredients(
	ctx context.Context,
	api http.Handler,
	args transferRecipe,
) (json.RawMessage, error) {
	selection := args.IngredientIndices
	if args.Portions < 0 || args.Portions > 1<<53-1 || (selection != nil && len(selection) == 0) {
		return nil, errors.New("provide positive portions and select at least one ingredient")
	}

	requestID, err := uuid.Parse(itemUUID(args.RequestID))
	if err != nil || requestID == uuid.Nil {
		return nil, errors.New("requestId must be a nonzero UUID")
	}

	data, err := callAPI(ctx, api, http.MethodGet, path("recipes", args.RecipeID), nil)
	if err != nil {
		return nil, err
	}

	var recipe recipeDraft
	if err := json.Unmarshal(data, &recipe); err != nil {
		return nil, err
	}

	for _, index := range selection {
		if index < 0 || index >= len(recipe.Ingredients) {
			return nil, fmt.Errorf("ingredient index %d does not exist", index)
		}
	}

	items := make([]itemInput, 0, len(recipe.Ingredients))
	for i, ingredient := range recipe.Ingredients {
		if selection != nil && !slices.Contains(selection, i) {
			continue
		}

		if args.Portions > 0 && args.Portions != recipe.Portions && ingredient.Amount != nil {
			amount, err := scaleAmount(*ingredient.Amount, args.Portions, recipe.Portions)
			if err != nil {
				return nil, err
			}

			ingredient.Amount = &amount
		}

		items = append(items, itemInput{
			ID:   uuid.NewSHA1(requestID, []byte(args.RecipeID+":"+strconv.Itoa(i))).String(),
			Text: ingredient.Name, Amount: ingredient.Amount, Unit: ingredient.Unit,
		})
	}

	return callAPI(ctx, api, http.MethodPost, listPath(args.ListID, "items", "batch"), items)
}

func scaleAmount(amount string, target, base int64) (string, error) {
	value, ok := new(big.Rat).SetString(amount)
	if !ok || value.Sign() <= 0 || base <= 0 {
		return "", errors.New("recipe has an invalid ingredient amount")
	}

	value.Mul(value, big.NewRat(target, base))

	precision, exact := value.FloatPrec()
	if !exact {
		precision = 6

		magnitude := new(big.Int).Mul(value.Num(), big.NewInt(10))
		for magnitude.Cmp(value.Denom()) < 0 {
			magnitude.Mul(magnitude, big.NewInt(10))

			precision++
		}
	}

	decimal := value.FloatString(precision)
	if precision > 0 {
		decimal = strings.TrimRight(strings.TrimRight(decimal, "0"), ".")
	}

	return decimal, nil
}
