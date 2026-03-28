package envutil

// ConvertToEnvKey converts a key name to UPPER_SNAKE_CASE environment variable format.
// Non-alphanumeric characters are replaced with underscores.
func ConvertToEnvKey(key string) string {
	result := make([]byte, 0, len(key))

	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'a' && c <= 'z' {
				c = c - 32
			}
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}

	return string(result)
}
