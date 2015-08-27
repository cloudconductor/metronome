package util

import "encoding/json"

type UnmarshalContext struct {
	Err error
}

func (u *UnmarshalContext) Unmarshal(data []byte, v interface{}) error {
	if u.Err != nil || len(data) == 0 {
		return u.Err
	}
	u.Err = json.Unmarshal(data, v)
	return u.Err
}
