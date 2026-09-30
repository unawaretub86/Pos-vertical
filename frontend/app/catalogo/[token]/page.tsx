import CatalogClient from './CatalogClient';

export function generateStaticParams() {
  return [{ token: '_' }];
}

export default function Page({ params }: { params: { token: string } }) {
  return <CatalogClient token={params.token} />;
}
