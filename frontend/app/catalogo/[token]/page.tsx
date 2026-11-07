import CatalogClient from './CatalogClient';

export function generateStaticParams() {
  return [{ token: '_' }];
}

// The token is read on the client (CatalogClient): the export bakes only the
// `_` placeholder, and Next 16 passes `params` as a Promise (US-086).
export default function Page() {
  return <CatalogClient />;
}
