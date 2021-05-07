package utils

func MustString(s string, err error) string {
	if err != nil {
		panic(err)
	} else {
		return s
	}
}
