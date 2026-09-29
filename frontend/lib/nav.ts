export interface NavFlags {
  adminOnly?: boolean;
  restaurantOnly?: boolean;
}

// Keeps the original order; hides admin-only items from cashiers and
// restaurant-only items while restaurant mode is off.
export function filterNavItems<T extends NavFlags>(
  items: T[],
  opts: { isAdmin: boolean; restaurantMode: boolean },
): T[] {
  return items.filter(
    item => (!item.adminOnly || opts.isAdmin) && (!item.restaurantOnly || opts.restaurantMode),
  );
}
