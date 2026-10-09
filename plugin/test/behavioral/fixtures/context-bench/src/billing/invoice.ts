export interface InvoiceLine {
  sku: string;
  quantity: number;
  unitPrice: number;
}

export function lineTotal(line: InvoiceLine): number {
  return line.quantity * line.unitPrice;
}

export function invoiceTotal(lines: InvoiceLine[]): number {
  return lines.reduce((sum, line) => sum + lineTotal(line), 0);
}
