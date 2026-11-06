// US-080 contract section 4, T-18 (AC-1, AC-2).
//
// Helper API assumed from the contract (names not given):
//   type PrinterForm = { print_method, paper_width_mm, printer_host, printer_port, printer_name }
//   buildPrinterPayload(form: PrinterForm): PrinterForm            // always carries all 5 fields
//   isTestPrintDisabled(saved: PrinterForm, form: PrinterForm): boolean  // true while unsaved printer changes exist
import { describe, it, expect } from 'vitest';
import { buildPrinterPayload, isTestPrintDisabled, type PrinterForm } from './printerSettings';

const network: PrinterForm = {
  print_method: 'escpos_network',
  paper_width_mm: 80,
  printer_host: '192.168.1.50',
  printer_port: 9100,
  printer_name: '',
};

describe('buildPrinterPayload', () => {
  it('escpos_network includes host, port and width', () => {
    expect(buildPrinterPayload(network)).toEqual({
      print_method: 'escpos_network',
      paper_width_mm: 80,
      printer_host: '192.168.1.50',
      printer_port: 9100,
      printer_name: '',
    });
  });

  it('browser still round-trips the stored host, port and name unchanged', () => {
    const form: PrinterForm = {
      print_method: 'browser',
      paper_width_mm: 58,
      printer_host: '192.168.1.50',
      printer_port: 9100,
      printer_name: 'POS-58',
    };
    expect(buildPrinterPayload(form)).toEqual(form);
  });
});

describe('isTestPrintDisabled', () => {
  it('is enabled when the form matches the saved config', () => {
    expect(isTestPrintDisabled(network, { ...network })).toBe(false);
  });

  it.each([
    ['method', { print_method: 'escpos_windows' }],
    ['width', { paper_width_mm: 58 }],
    ['host', { printer_host: '192.168.1.51' }],
    ['port', { printer_port: 9101 }],
    ['name', { printer_name: 'POS-58' }],
  ] as const)('is disabled after an unsaved %s change', (_label, change) => {
    expect(isTestPrintDisabled(network, { ...network, ...change })).toBe(true);
  });
});
