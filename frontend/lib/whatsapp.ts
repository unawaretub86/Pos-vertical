import { formatCOP } from './format';

/**
 * Collection (debt reminder) WhatsApp message shared by the customer detail
 * "Cobrar" button and the Customers list link. An empty or whitespace-only
 * business name (not configured, config still loading, or GET /setup failed)
 * drops the business sentence instead of using a filler.
 */
export function buildCollectionMessage(customerName: string, balance: number, businessName: string): string {
  const amount = formatCOP(balance);
  const biz = businessName.trim();
  if (!biz) {
    return `Hola ${customerName}, le recordamos que tiene un saldo pendiente de ${amount}. Gracias.`;
  }
  return `Hola ${customerName}, le escribimos de ${biz}. Le recordamos que tiene un saldo pendiente de ${amount}. Gracias.`;
}

/**
 * wa.me URL for the collection message. The phone keeps its digits only and
 * gets the Colombian 57 prefix unless it already starts with 57.
 */
export function buildCollectionWhatsAppUrl(phone: string, customerName: string, balance: number, businessName: string): string {
  const digits = phone.replace(/\D/g, '');
  const fullPhone = digits.startsWith('57') ? digits : `57${digits}`;
  const text = buildCollectionMessage(customerName, balance, businessName);
  return `https://wa.me/${fullPhone}?text=${encodeURIComponent(text)}`;
}
