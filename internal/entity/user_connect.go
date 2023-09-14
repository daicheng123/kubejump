package entity

type ConnectInfo struct {
	Id    string `json:"id"`
	User  *User  `json:"user"`
	Value string `json:"value"`
	//Account  Account    `json:"account"`
	//Actions  Actions    `json:"actions"`
	Asset *Asset `json:"asset"`
	//Protocol string `json:"protocol"`
	//Domain   *Domain    `json:"domain"`
	//Gateway  *Gateway   `json:"gateway"`
	ExpireAt ExpireInfo `json:"expire_at"`
	//OrgId    string     `json:"org_id"`
	//OrgName  string     `json:"org_name"`
	//Platform Platform   `json:"platform"`

	ConnectOptions ConnectOptions `json:"connect_options"`

	//CommandFilterACLs []CommandACL `json:"command_filter_acls"`
	//
	//Ticket     *ObjectId   `json:"from_ticket,omitempty"`
	TicketInfo interface{} `json:"from_ticket_info,omitempty"`

	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type ConnectOptions struct {
	Charset          *string `json:"charset,omitempty"`
	DisableAutoHash  *bool   `json:"disableautohash,omitempty"`
	BackspaceAsCtrlH *bool   `json:"backspaceAsCtrlH,omitempty"`
}
