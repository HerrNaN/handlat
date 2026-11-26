import z from "zod";

import { readApiError } from "../api";
import { clearSession } from "../user/session";
import { ItemSchema, type Item } from "./list";

export const ListSchema = z.object({
  id: z.string(),
  name: z.string(),
  items: z.array(ItemSchema),
  memberCount: z.number().int().positive(),
});
const AddItemResultSchema = ListSchema.extend({ mergedItemId: z.string().optional() });
const SummariesSchema = z.array(
  z.object({
    id: z.string(),
    name: z.string(),
    itemCount: z.number().int(),
    memberCount: z.number().int().positive(),
    isDefault: z.boolean().default(false),
  }),
);

export type ShoppingList = z.infer<typeof ListSchema>;
export type AddItemResult = z.infer<typeof AddItemResultSchema>;
export type ListSummary = z.infer<typeof SummariesSchema>[number];
export type NewItem = Pick<Item, "id" | "text" | "department" | "amount" | "unit">;
export type ItemChanges = Partial<Pick<Item, "text" | "department">> & {
  amount?: string | null;
  unit?: Item["unit"] | null;
};

const request = async (url: string, init?: RequestInit) => {
  const response = await fetch(url, { cache: "no-store", ...init });
  if (!response.ok) {
    if (response.status === 401) {
      clearSession();
    }
    throw await readApiError(response);
  }
  return response.status === 204 ? undefined : response.json();
};

const listURL = (id: string) => `/api/lists/${encodeURIComponent(id)}`;
const itemURL = (listID: string, itemID: string) =>
  `${listURL(listID)}/items/${encodeURIComponent(itemID)}`;
const jsonBody = (body: unknown) => ({
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});
const requestList = async (url: string, init?: RequestInit) =>
  ListSchema.parse(await request(url, init));

export const getLists = async (signal?: AbortSignal) =>
  SummariesSchema.parse(await request("/api/lists", { signal }));
export const setDefaultShoppingList = async (listId: string): Promise<void> => {
  await request("/api/lists/default", { method: "PUT", ...jsonBody({ listId }) });
};
export const getList = (id: string, signal?: AbortSignal) => requestList(listURL(id), { signal });
export const createList = (name: string) =>
  requestList("/api/lists", { method: "POST", ...jsonBody({ name }) });
export const renameList = (id: string, name: string) =>
  requestList(listURL(id), { method: "PATCH", ...jsonBody({ name }) });
export const leaveList = async (id: string): Promise<void> => {
  await request(listURL(id), { method: "DELETE" });
};
export const addItem = async (listID: string, item: NewItem) =>
  AddItemResultSchema.parse(
    await request(`${listURL(listID)}/items`, { method: "POST", ...jsonBody(item) }),
  );
export const addItems = (listID: string, items: NewItem[]) =>
  requestList(`${listURL(listID)}/items/batch`, { method: "POST", ...jsonBody(items) });
export const updateItem = (listID: string, itemID: string, changes: ItemChanges) =>
  requestList(itemURL(listID, itemID), { method: "PATCH", ...jsonBody(changes) });
export const setItemChecked = (listID: string, itemID: string, checked: boolean) =>
  requestList(itemURL(listID, itemID), { method: "PATCH", ...jsonBody({ checked }) });
export const deleteItem = (listID: string, itemID: string) =>
  requestList(itemURL(listID, itemID), { method: "DELETE" });
export const clearCheckedItems = (listID: string) =>
  requestList(`${listURL(listID)}/checked-items`, { method: "DELETE" });

const invitationURL = (token: string) => `/api/invitations/${encodeURIComponent(token)}`;

export const createInvitation = async (listID: string) =>
  z
    .object({ token: z.string(), expiresAt: z.string() })
    .parse(await request(`${listURL(listID)}/invitations`, { method: "POST" }));
export const getInvitation = async (token: string, signal?: AbortSignal) =>
  z.object({ name: z.string() }).parse(await request(invitationURL(token), { signal }));
export const acceptInvitation = async (token: string) =>
  ListSchema.parse(await request(`${invitationURL(token)}/accept`, { method: "POST" }));
