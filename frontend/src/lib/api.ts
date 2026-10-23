export const post = async (url: string, body?: unknown) => {
  const response = await fetch(url, {
    method: "POST",
    ...(body === undefined ? {} : {
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  });
  if (!response.ok) throw new Error((await response.text()) || "Request failed");
  return response;
};
