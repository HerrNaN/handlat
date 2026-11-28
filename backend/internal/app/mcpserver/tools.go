package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listID struct {
	ListID string `json:"listId" jsonschema:"ID of a shopping list returned by list_shopping_lists"`
}

type listName struct {
	Name string `json:"name" jsonschema:"Shopping-list name"`
}

type renameList struct {
	listID
	listName
}

type itemID struct {
	listID
	ItemID string `json:"itemId" jsonschema:"ID of an item on the list"`
}

type itemInput struct {
	ID         string  `json:"id,omitempty"         jsonschema:"Optional UUID. Reuse the same UUID when retrying an addition."`
	Text       string  `json:"text"`
	Department string  `json:"department,omitempty"`
	Amount     *string `json:"amount,omitempty"     jsonschema:"Positive decimal as a string"`
	Unit       *string `json:"unit,omitempty"       jsonschema:"One of st, pkt, kg, g, l, dl, cl, ml, msk, tsk, krm"`
}

type addItem struct {
	listID
	itemInput
}

type addItems struct {
	listID
	Items []itemInput `json:"items"`
}

type updateItem struct {
	itemID
	Text        *string `json:"text,omitempty"`
	Department  *string `json:"department,omitempty"`
	Checked     *bool   `json:"checked,omitempty"`
	Amount      *string `json:"amount,omitempty"`
	Unit        *string `json:"unit,omitempty"`
	ClearAmount bool    `json:"clearAmount,omitempty"`
	ClearUnit   bool    `json:"clearUnit,omitempty"`
}

type invitation struct {
	Token string `json:"token" jsonschema:"Token from a Handlat shopping-list invitation link"`
}

type recipeID struct {
	RecipeID string `json:"recipeId" jsonschema:"ID of a private recipe returned by list_recipes"`
}

type ingredient struct {
	Name   string  `json:"name"`
	Amount *string `json:"amount,omitempty" jsonschema:"Positive decimal as a string"`
	Unit   *string `json:"unit,omitempty"   jsonschema:"One of st, pkt, kg, g, l, dl, cl, ml, msk, tsk, krm"`
}

type recipeDraft struct {
	Name        string       `json:"name"`
	Portions    int64        `json:"portions"`
	Ingredients []ingredient `json:"ingredients"`
}

type createRecipe struct {
	ID string `json:"id,omitempty" jsonschema:"Optional UUID. Reuse it when retrying the creation."`
	recipeDraft
}

type updateRecipe struct {
	recipeID
	recipeDraft
}

type transferRecipe struct {
	listID
	recipeID
	Portions          int64  `json:"portions,omitempty"          jsonschema:"Target portions. Omit to use the recipe's original portions."`
	IngredientIndices []int  `json:"ingredientIndices,omitempty" jsonschema:"Zero-based ingredient indices. Omit to add all ingredients."`
	RequestID         string `json:"requestId,omitempty"         jsonschema:"Optional UUID. Reuse it when retrying the transfer."`
}

func newToolHandler(api http.Handler, origin, version string) http.Handler {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "handlat", Title: "Handlat", Version: version},
		nil,
	)
	registerListTools(server, api, origin)
	registerRecipeTools(server, api)

	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			Stateless: true, JSONResponse: true,
			// Handlat checks the Origin and bearer token before this handler.
			// A local reverse proxy can deliver requests for the public origin.
			DisableLocalhostProtection: true,
		},
	)
}

func registerListTools(server *mcp.Server, api http.Handler, origin string) {
	addTool(
		server,
		"list_shopping_lists",
		"Find the user's shopping lists and their IDs.",
		true,
		false,
		func(ctx context.Context, _ struct{}) (any, error) {
			return callAPI(ctx, api, http.MethodGet, "/lists", nil)
		},
	)
	addTool(server, "get_shopping_list", "Read a shopping list and its items.", true, false,
		func(ctx context.Context, args listID) (any, error) {
			return callAPI(ctx, api, http.MethodGet, listPath(args.ListID), nil)
		})
	addTool(server, "create_shopping_list", "Create a new shopping list.", false, false,
		func(ctx context.Context, args listName) (any, error) {
			return callAPI(ctx, api, http.MethodPost, "/lists", args)
		})
	addTool(
		server,
		"set_default_shopping_list",
		"Set the user's default shopping list.",
		false,
		true,
		func(ctx context.Context, args listID) (any, error) {
			return callAPI(ctx, api, http.MethodPut, "/lists/default", args)
		},
	)
	addTool(server, "rename_shopping_list", "Replace a shopping list's name.", false, true,
		func(ctx context.Context, args renameList) (any, error) {
			return callAPI(ctx, api, http.MethodPatch, listPath(args.ListID), args.listName)
		})
	addTool(
		server,
		"leave_shopping_list",
		"Leave a shopping list. If the user is its last member, delete the list and its items.",
		false,
		true,
		func(ctx context.Context, args listID) (any, error) {
			return callAPI(ctx, api, http.MethodDelete, listPath(args.ListID), nil)
		},
	)
	addTool(
		server,
		"add_shopping_list_item",
		"Add an item, or merge its quantity with a matching unchecked item.",
		false,
		false,
		func(ctx context.Context, args addItem) (any, error) {
			args.ID = itemUUID(args.ID)

			return callAPI(
				ctx,
				api,
				http.MethodPost,
				listPath(args.ListID, "items"),
				args.itemInput,
			)
		},
	)
	addTool(
		server,
		"add_shopping_list_items",
		"Add multiple items to one shopping list atomically.",
		false,
		false,
		func(ctx context.Context, args addItems) (any, error) {
			for i := range args.Items {
				args.Items[i].ID = itemUUID(args.Items[i].ID)
			}

			return callAPI(
				ctx,
				api,
				http.MethodPost,
				listPath(args.ListID, "items", "batch"),
				args.Items,
			)
		},
	)
	addTool(
		server,
		"update_shopping_list_item",
		"Edit an item, clear its quantity fields, or check or uncheck it.",
		false,
		true,
		func(ctx context.Context, args updateItem) (any, error) {
			body := map[string]any{}
			if args.Text != nil {
				body["text"] = args.Text
			}

			if args.Department != nil {
				body["department"] = args.Department
			}

			if args.Checked != nil {
				body["checked"] = args.Checked
			}

			if args.Amount != nil || args.ClearAmount {
				body["amount"] = args.Amount
			}

			if args.Unit != nil || args.ClearUnit {
				body["unit"] = args.Unit
			}

			if (args.ClearAmount && args.Amount != nil) || (args.ClearUnit && args.Unit != nil) {
				return nil, errors.New("set or clear a quantity field in one call")
			}

			return callAPI(
				ctx,
				api,
				http.MethodPatch,
				listPath(args.ListID, "items", args.ItemID),
				body,
			)
		},
	)
	addTool(server, "delete_shopping_list_item", "Delete a shopping-list item.", false, true,
		func(ctx context.Context, args itemID) (any, error) {
			return callAPI(
				ctx,
				api,
				http.MethodDelete,
				listPath(args.ListID, "items", args.ItemID),
				nil,
			)
		})
	addTool(
		server,
		"clear_checked_items",
		"Delete all currently checked items on a shopping list.",
		false,
		true,
		func(ctx context.Context, args listID) (any, error) {
			return callAPI(ctx, api, http.MethodDelete, listPath(args.ListID, "checked-items"), nil)
		},
	)
	addTool(
		server,
		"create_list_invitation",
		"Create a link that lets its recipient join the shopping list.",
		false,
		false,
		func(ctx context.Context, args listID) (any, error) {
			data, err := callAPI(
				ctx,
				api,
				http.MethodPost,
				listPath(args.ListID, "invitations"),
				nil,
			)
			if err != nil {
				return nil, err
			}

			var invite struct {
				Token     string `json:"token"`
				ExpiresAt string `json:"expiresAt"`
			}
			if err := json.Unmarshal(data, &invite); err != nil {
				return nil, err
			}

			return map[string]string{
				"token": invite.Token, "expiresAt": invite.ExpiresAt,
				"url": origin + path("invitations", invite.Token),
			}, nil
		},
	)
	addTool(
		server,
		"get_list_invitation",
		"Read the shopping-list name from an invitation token.",
		true,
		false,
		func(ctx context.Context, args invitation) (any, error) {
			return callAPI(ctx, api, http.MethodGet, path("invitations", args.Token), nil)
		},
	)
	addTool(
		server,
		"accept_list_invitation",
		"Accept an invitation and join the shopping list.",
		false,
		false,
		func(ctx context.Context, args invitation) (any, error) {
			return callAPI(
				ctx,
				api,
				http.MethodPost,
				path("invitations", args.Token, "accept"),
				nil,
			)
		},
	)
}

func registerRecipeTools(server *mcp.Server, api http.Handler) {
	addTool(
		server,
		"list_recipes",
		"Find the user's private recipes. Also shows whether recipes are enabled.",
		true,
		false,
		func(ctx context.Context, _ struct{}) (any, error) {
			return callAPI(ctx, api, http.MethodGet, "/recipes", nil)
		},
	)
	addTool(
		server,
		"enable_recipes",
		"Enable the private recipes feature for this account.",
		false,
		false,
		func(ctx context.Context, _ struct{}) (any, error) {
			return callAPI(ctx, api, http.MethodPost, "/recipes/enable", nil)
		},
	)
	addTool(server, "get_recipe", "Read a private recipe and its ingredients.", true, false,
		func(ctx context.Context, args recipeID) (any, error) {
			return callAPI(ctx, api, http.MethodGet, path("recipes", args.RecipeID), nil)
		})
	addTool(server, "create_recipe", "Create a private recipe with ingredients.", false, false,
		func(ctx context.Context, args createRecipe) (any, error) {
			args.ID = itemUUID(args.ID)
			return callAPI(ctx, api, http.MethodPost, "/recipes", args)
		})
	addTool(
		server,
		"update_recipe",
		"Replace a private recipe's name, portions, and ingredients.",
		false,
		true,
		func(ctx context.Context, args updateRecipe) (any, error) {
			return callAPI(
				ctx,
				api,
				http.MethodPut,
				path("recipes", args.RecipeID),
				args.recipeDraft,
			)
		},
	)
	addTool(server, "delete_recipe", "Permanently delete a private recipe.", false, true,
		func(ctx context.Context, args recipeID) (any, error) {
			return callAPI(ctx, api, http.MethodDelete, path("recipes", args.RecipeID), nil)
		})
	addTool(
		server,
		"transfer_recipe_ingredients",
		"Add ingredients from a private recipe to a shopping list. Supports selecting ingredients and scaling portions.",
		false,
		false,
		func(ctx context.Context, args transferRecipe) (any, error) {
			return transferIngredients(ctx, api, args)
		},
	)
}

func addTool[Args any](
	server *mcp.Server,
	name, description string,
	readOnly, destructive bool,
	call func(context.Context, Args) (any, error),
) {
	mcp.AddTool(server, &mcp.Tool{
		Name: name, Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: readOnly, DestructiveHint: &destructive, OpenWorldHint: new(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error) {
		result, err := call(ctx, args)
		if err != nil {
			return nil, nil, err
		}

		return nil, map[string]any{"result": result}, nil
	})
}

func itemUUID(value string) string {
	if value == "" {
		return uuid.NewString()
	}

	return value
}

func listPath(id string, segments ...string) string {
	return path(append([]string{"lists", id}, segments...)...)
}

func path(segments ...string) string {
	var value strings.Builder
	for _, segment := range segments {
		value.WriteByte('/')
		value.WriteString(url.PathEscape(segment))
	}

	return value.String()
}

type apiResponse struct {
	body   bytes.Buffer
	header http.Header
	status int
}

func (r *apiResponse) Header() http.Header { return r.header }
func (r *apiResponse) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

func (r *apiResponse) Write(data []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	return r.body.Write(data)
}

func callAPI(
	ctx context.Context,
	api http.Handler,
	method, path string,
	body any,
) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response := &apiResponse{header: make(http.Header)}
	api.ServeHTTP(response, request)

	if response.status < 200 || response.status >= 300 {
		var failure struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(response.body.Bytes(), &failure) == nil && failure.Code != "" {
			return nil, fmt.Errorf("operation rejected: %s", failure.Code)
		}

		return nil, fmt.Errorf("operation rejected (HTTP %d)", response.status)
	}

	if response.body.Len() == 0 {
		return json.RawMessage(`{"ok":true}`), nil
	}

	return json.RawMessage(response.body.Bytes()), nil
}
