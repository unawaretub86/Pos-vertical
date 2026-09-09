package factelect

// unitToDIAN maps product unit field to DIAN unit code.
// Default "94" (unit) for unknown values.
var unitToDIAN = map[string]string{
	"unit":   "94",
	"und":    "94",
	"unidad": "94",
	"kg":     "KGM",
	"liter":  "LTR",
	"lb":     "LBR",
	"m":      "MTR",
	"m2":     "MTK",
	"m3":     "MTQ",
	"caja":   "BX",
	"par":    "PR",
	"hora":   "HUR",
}

// UnitCode returns the DIAN code for a unit of measure.
func UnitCode(unit string) string {
	if code, ok := unitToDIAN[unit]; ok {
		return code
	}
	return "94"
}

// paymentMethodCode maps POS payment_method to DIAN code.
// "10"=cash, "20"=transfer, "42"=debit, "48"=credit.
var paymentMethodCode = map[string]string{
	"cash":     "10",
	"transfer": "20",
	"card":     "42",
	"credit":   "48",
}

// PaymentCode returns the DIAN code for a payment method.
func PaymentCode(method string) string {
	if code, ok := paymentMethodCode[method]; ok {
		return code
	}
	return "10"
}

// productTypeCode returns the DIAN item type based on product type.
// "01"=merchandise, "04"=service.
func productTypeCode(productType string) string {
	if productType == "service" {
		return "04"
	}
	return "01"
}
