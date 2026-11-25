export const formatIngredientName = (name: string) =>
  name.replace(/^./u, (first) => first.toUpperCase());
