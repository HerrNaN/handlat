import z from "zod";
import { clearSession } from "../user/session";
import { ItemSchema, type Item } from "./list";

const ListSchema = z.object({ id: z.string(), name: z.string(), items: z.array(ItemSchema) });
const SummariesSchema = z.array(z.object({ id: z.string(), name: z.string(), itemCount: z.number().int() }));

export type ShoppingList = z.infer<typeof ListSchema>;
export type ListSummary = z.infer<typeof SummariesSchema>[number];
export type NewItem = Pick<Item, "id" | "text" | "department">;
export type ItemChanges = Partial<Pick<Item, "text" | "department">>;

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

export const errorMessage = (error: unknown) => error instanceof ApiError ? error.message
  : "Unable to reach the server. Lists require an internet connection. Refresh to check server state before retrying your change.";

const request = async (url: string, init?: RequestInit) => {
  const response = await fetch(url, { cache: "no-store", ...init });
  if (!response.ok) {
    if (response.status === 401) clearSession();
    throw new ApiError(response.status,
      response.status === 401 ? "Your session expired. Please sign in again."
      : response.status === 404 ? "This list or item is no longer available."
      : response.status >= 500 ? "The server could not save or load your list. Try again."
      : (await response.text()) || "The request failed.");
  }
  return response.json();
};

const listURL = (id: string) => `/api/lists/${encodeURIComponent(id)}`;
const itemURL = (listID: string, itemID: string) => `${listURL(listID)}/items/${encodeURIComponent(itemID)}`;
const jsonBody = (body: unknown) => ({ headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const requestList = async (url: string, init?: RequestInit) => ListSchema.parse(await request(url, init));

export const getLists = async (signal?: AbortSignal) => SummariesSchema.parse(await request("/api/lists", { signal }));
export const getList = (id: string, signal?: AbortSignal) => requestList(listURL(id), { signal });
export const createList = (name: string) => requestList("/api/lists", { method: "POST", ...jsonBody({ name }) });
export const renameList = (id: string, name: string) => requestList(listURL(id), { method: "PATCH", ...jsonBody({ name }) });
export const addItem = (listID: string, item: NewItem) => requestList(`${listURL(listID)}/items`, { method: "POST", ...jsonBody(item) });
export const updateItem = (listID: string, itemID: string, changes: ItemChanges) => requestList(itemURL(listID, itemID), { method: "PATCH", ...jsonBody(changes) });
export const setItemChecked = (listID: string, itemID: string, checked: boolean) => requestList(itemURL(listID, itemID), { method: "PATCH", ...jsonBody({ checked }) });
export const deleteItem = (listID: string, itemID: string) => requestList(itemURL(listID, itemID), { method: "DELETE" });
export const clearCheckedItems = (listID: string) => requestList(`${listURL(listID)}/checked-items`, { method: "DELETE" });
