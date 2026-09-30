import OrderClient from './OrderClient';

export function generateStaticParams() {
  return [{ orderId: '_' }];
}

export default function Page() {
  return <OrderClient />;
}
