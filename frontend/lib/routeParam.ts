// Dynamic route segments for the static export (US-086).
//
// `output: "export"` pre-renders each dynamic route only as `_`, and the Go
// server serves that one page for every `/<prefix>/*` URL. In the binary the
// route param is therefore `_` (or `undefined`), so the real segment has to be
// read from the browser path. Under `pnpm dev` the param is real and wins.

const PLACEHOLDER = '_';

function isRealSegment(value: string): boolean {
  return value !== '' && value !== PLACEHOLDER;
}

// Returns the route segment for `/<prefix>/<segment>`: the route param when it
// is a real value, otherwise the URI-decoded first segment after the prefix in
// `pathname`. Returns null when there is no usable segment; never throws.
export function resolveRouteParam(
  param: string | string[] | undefined,
  pathname: string,
  prefix: string,
): string | null {
  if (typeof param === 'string' && isRealSegment(param)) return param;

  const head = `/${prefix}/`;
  if (!pathname.startsWith(head)) return null;
  const raw = pathname.slice(head.length).split('/')[0];
  if (raw === undefined) return null;

  let segment: string;
  try {
    segment = decodeURIComponent(raw);
  } catch {
    return null;
  }
  return isRealSegment(segment) ? segment : null;
}

// Customer ids are positive integers; anything else means "no customer".
export function parseCustomerId(segment: string | null): number | null {
  if (segment === null || !/^[1-9]\d*$/.test(segment)) return null;
  const id = Number(segment);
  return Number.isSafeInteger(id) ? id : null;
}
