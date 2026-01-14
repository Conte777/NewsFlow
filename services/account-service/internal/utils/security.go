package utils

// MaskPhoneNumber masks a phone number for secure logging
// Keeps first 3 and last 4 characters visible, masks the rest
//
// Examples:
//   - "+1234567890" -> "+12****7890"
//   - "+123456" -> "****"
//   - "short" -> "****"
func MaskPhoneNumber(phone string) string {
	if len(phone) <= 6 {
		return "****"
	}

	// Keep first 3 and last 4 digits, mask the rest
	return phone[:3] + "****" + phone[len(phone)-4:]
}
