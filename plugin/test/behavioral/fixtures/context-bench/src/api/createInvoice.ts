import { invoiceTotal, type InvoiceLine } from '../billing/invoice';

export function createInvoiceHandler(body: { lines: InvoiceLine[] }): { total: number } {
  return { total: invoiceTotal(body.lines) };
}
