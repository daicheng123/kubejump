package entity

import "encoding/json"

type LabelField string

func (s LabelField) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s + `"`), nil
}

func (s *LabelField) UnmarshalJSON(bytes []byte) error {
	var labelValue struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(bytes, &labelValue); err != nil {
		return err
	}
	*s = LabelField(labelValue.Value)
	return nil
}

type Session struct {
	ID         string         `json:"id,omitempty"`
	User       string         `json:"user"`
	Asset      string         `json:"asset"`
	Account    string         `json:"account"`
	LoginFrom  LabelField     `json:"login_from,omitempty"`
	RemoteAddr string         `json:"remote_addr"`
	Protocol   string         `json:"protocol"`
	DateStart  common.UTCTime `json:"date_start"`
	OrgID      string         `json:"org_id"`
	UserID     string         `json:"user_id"`
	AssetID    string         `json:"asset_id"`
	AccountID  string         `json:"account_id"`
	Type       LabelField     `json:"type"`
}
