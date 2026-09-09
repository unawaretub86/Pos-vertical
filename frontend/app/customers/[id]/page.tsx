import CustomerDetailPage from './CustomerDetailClient';

export function generateStaticParams() {
  return [{ id: '_' }];
}

export default function Page() {
  return <CustomerDetailPage />;
}
