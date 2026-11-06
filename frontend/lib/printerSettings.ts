// US-080: printer fields of the Settings form.

export interface PrinterForm {
  print_method: string;
  paper_width_mm: number;
  printer_host: string;
  printer_port: number;
  printer_name: string;
}

// PUT /setup rewrites every column, so the payload always carries all five
// printer fields, even the ones hidden by the chosen method.
export function buildPrinterPayload(form: PrinterForm): PrinterForm {
  return {
    print_method: form.print_method,
    paper_width_mm: form.paper_width_mm,
    printer_host: form.printer_host,
    printer_port: form.printer_port,
    printer_name: form.printer_name,
  };
}

// The test print uses the SAVED config, so it is disabled while the form
// differs from it.
export function isTestPrintDisabled(saved: PrinterForm, form: PrinterForm): boolean {
  return (
    saved.print_method !== form.print_method ||
    saved.paper_width_mm !== form.paper_width_mm ||
    saved.printer_host !== form.printer_host ||
    saved.printer_port !== form.printer_port ||
    saved.printer_name !== form.printer_name
  );
}
