export function formatPrice(amount: number, currency: string): string {
  const rounded = Math.round(amount * 100) / 100;
  const [whole, cents = ''] = rounded.toFixed(2).split('.');
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
  return `${grouped},${cents} ${currency.toUpperCase()}`;
}
