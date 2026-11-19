import { ApiError } from "../api";
import type { Messages } from "./en";

export const errorMessage = (error: unknown, messages: Messages, fallback: string): string => {
  const { errors } = messages;
  if (!(error instanceof ApiError)) return fallback;
  if (error.status === 401) return errors.sessionExpired;

  const { min, max } = error.params ?? {};
  switch (error.code) {
    case "invalid_list_name":
      return min !== undefined && max !== undefined ? errors.listNameLength(min, max) : fallback;
    case "invalid_item_text":
      return min !== undefined && max !== undefined ? errors.itemNameLength(min, max) : fallback;
    case "invalid_department":
      return min !== undefined && max !== undefined ? errors.departmentLength(min, max) : fallback;
    case "invalid_amount":
      return max !== undefined ? errors.invalidAmount(max) : fallback;
    case "unsupported_unit":
      return errors.unsupportedUnit;
    case "list_or_item_unavailable":
      return errors.listUnavailable;
    case "invitation_unavailable":
      return errors.invitationUnavailable;
    case "sign_in_expired":
      return errors.signInExpired;
    case "registration_expired":
      return errors.registrationExpired;
    case "passkey_verification_failed":
      return errors.passkeyVerificationFailed;
    default:
      return fallback;
  }
};

export const passkeyErrorMessage = (error: unknown, messages: Messages, fallback: string) => {
  const cause: unknown = error instanceof Error ? error.cause : undefined;
  if (
    (error instanceof DOMException && error.name === "NotAllowedError") ||
    (cause instanceof DOMException && cause.name === "NotAllowedError")
  ) {
    return messages.errors.passkeyCancelled;
  }
  return errorMessage(error, messages, fallback);
};
