package api

// API is the interface for different panel's api.
type API interface {
	GetNodeInfo() (nodeInfo *NodeInfo, err error)
	GetUserList() (userList []*UserInfo, err error)
	ReportUserTraffic(userTraffic []*UserTraffic) (err error)
	Describe() *ClientInfo
	Debug()
}

type UserTraffic struct {
	UID      int   `json:"user_id"`
	Upload   int64 `json:"u"`
	Download int64 `json:"d"`
}

type RepUserTraffic struct {
	Message string `json:"message"`
}

type NodeInfo struct {
	ID            int    `json:"id"`
	ServerPort    int    `json:"server_port"`
	AllowInsecure int    `json:"allow_insecure"`
	ServerName    string `json:"server_name"`
}

type RepNodeInfo struct {
	Data    *NodeInfo `json:"data"`
	Message string    `json:"message"`
}

type UserInfo struct {
	ID   int    `json:"id"`
	UUID string `json:"uuid"`
}

type RepUserList struct {
	Data    []*UserInfo `json:"data"`
	Message string      `json:"message"`
}

type ClientInfo struct {
	APIHost string
	NodeID  int
	Token   string
}
