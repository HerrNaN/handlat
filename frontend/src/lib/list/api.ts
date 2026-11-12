import z from "zod";

import { clearSession } from "../user/session";
import { ItemSchema, type Item } from "./list";

export const ListSchema = z.object({
  id: z.string(),
  name: z.string(),
  items: z.array(ItemSchema),
  memberCount: z.number().int().positive(),
});
const SummariesSchema = z.array(
  z.object({
    id: z.string(),
    name: z.string(),
    itemCount: z.number().int(),
    memberCount: z.number().int().positive(),
  }),
);

export type ShoppingList = z.infer<typeof ListSchema>;
export type ListSummary = z.infer<typeof SummariesSchema>[number];
export type NewItem = Pick<Item, "id" | "text" | "department" | "amount" | "unit">;
export type ItemChanges = Partial<Pick<Item, "text" | "department">> & {
  amount?: string | null;
  unit?: Item["unit"] | null;
};

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

export const errorMessage = (error: unknown) =>
  error instanceof ApiError ? error.message : "Unable to reach the server.";

const request = async (url: string, init?: RequestInit) => {
  const response = await fetch(url, { cache: "no-store", ...init });
  if (!response.ok) {
    if (response.status === 401) clearSession();
    throw new ApiError(
      response.status,
      response.status === 401
        ? "Your session expired. Please log in again."
        : response.status === 404
          ? "This list or item is no longer available."
          : response.status >= 500
            ? "The server could not save or load your list. Try again."
            : (await response.text()) || "The request failed.",
    );
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
export const getList = (id: string, signal?: AbortSignal) => requestList(listURL(id), { signal });
export const createList = (name: string) =>
  requestList("/api/lists", { method: "POST", ...jsonBody({ name }) });
export const renameList = (id: string, name: string) =>
  requestList(listURL(id), { method: "PATCH", ...jsonBody({ name }) });
export const leaveList = async (id: string): Promise<void> => {
  await request(listURL(id), { method: "DELETE" });
};
export const addItem = (listID: string, item: NewItem) =>
  requestList(`${listURL(listID)}/items`, { method: "POST", ...jsonBody(item) });
export const updateItem = (listID: string, itemID: string, changes: ItemChanges) =>
  requestList(itemURL(listID, itemID), { method: "PATCH", ...jsonBody(changes) });
export const setItemChecked = (listID: string, itemID: string, checked: boolean) =>
  requestList(itemURL(listID, itemID), { method: "PATCH", ...jsonBody({ checked }) });
export const deleteItem = (listID: string, itemID: string) =>
  requestList(itemURL(listID, itemID), { method: "DELETE" });
export const clearCheckedItems = (listID: string) =>
  requestList(`${listURL(listID)}/checked-items`, { method: "DELETE" });

const invitationURL = (token: string) => `/api/invitations/${encodeURIComponent(token)}`;
const invitationRequest = async (url: string, init?: RequestInit) => {
  try {
    return await request(url, init);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      throw new ApiError(404, "This invitation is no longer available.");
    }
    throw error;
  }
};

export const createInvitation = async (listID: string) =>
  z
    .object({ token: z.string(), expiresAt: z.string() })
    .parse(await request(`${listURL(listID)}/invitations`, { method: "POST" }));
export const getInvitation = async (token: string, signal?: AbortSignal) =>
  z.object({ name: z.string() }).parse(await invitationRequest(invitationURL(token), { signal }));
export const acceptInvitation = async (token: string) =>
  ListSchema.parse(await invitationRequest(`${invitationURL(token)}/accept`, { method: "POST" }));
