import z from "zod";

import { readApiError } from "../api";
import { AmountSchema, UnitSchema } from "../list/quantity";
import { clearSession } from "../user/session";

const summarySchema = z.object({ id: z.string(), name: z.string() });
const indexSchema = z.object({ enabled: z.boolean(), recipes: z.array(summarySchema) });
const recipeSchema = summarySchema.extend({
  portions: z.number().int().positive(),
  ingredients: z.array(
    z.object({ name: z.string(), amount: AmountSchema.optional(), unit: UnitSchema.optional() }),
  ),
});

export type Recipe = z.infer<typeof recipeSchema>;
export type RecipeDraft = Omit<Recipe, "id">;

const url = "/api/recipes";
const recipeURL = (id: string) => `${url}/${encodeURIComponent(id)}`;
const json = (body: unknown) => ({
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

const request = async (path: string, init?: RequestInit) => {
  const response = await fetch(path, { cache: "no-store", ...init });
  if (!response.ok) {
    if (response.status === 401) clearSession();
    throw await readApiError(response);
  }
  return response.status === 204 ? undefined : response.json();
};

export const recipesKey = (account?: string) => ["private-recipes", account, "index"] as const;
export const recipeKey = (account: string | undefined, id: string) =>
  ["private-recipes", account, "detail", id] as const;

export const getRecipes = async (signal?: AbortSignal) =>
  indexSchema.parse(await request(url, { signal }));
export const getRecipe = async (id: string, signal?: AbortSignal) =>
  recipeSchema.parse(await request(recipeURL(id), { signal }));
export const enableRecipes = () => request(`${url}/enable`, { method: "POST" });

export const saveRecipe = async (id: string, draft: RecipeDraft, create: boolean) =>
  recipeSchema.parse(
    await request(create ? url : recipeURL(id), {
      method: create ? "POST" : "PUT",
      ...json(create ? { id, ...draft } : draft),
    }),
  );

export const deleteRecipe = (id: string) => request(recipeURL(id), { method: "DELETE" });
