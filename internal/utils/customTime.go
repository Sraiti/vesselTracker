package utils

import "time"

type CustomTime struct {
	time.Time
}

/*
fixing parsing time zone error when using UnmarshalJSON() function
look at this github issue for more : https://github.com/go-swagger/go-swagger/issues/873
*/
func (ct *CustomTime) UnmarshalJSON(b []byte) error {

	s := string(b)
	s = s[1 : len(s)-1]

	// Parsing with different layouts/types of dates
	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	var firstErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			ct.Time = t
			return nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
