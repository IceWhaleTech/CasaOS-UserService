package model

type ServerModel struct {
	HttpPort string
	RunMode  string
}

type APPModel struct {
	LogPath        string
	LogSaveName    string
	LogFileExt     string
	DateStrFormat  string
	DateTimeFormat string
	UserDataPath   string
	TimeFormat     string
	DateFormat     string
	DBPath         string
	ShellPath      string
	TempPath       string
}

type Result struct {
	Success int         `json:"success" example:"200"`
	Message string      `json:"message" example:"ok"`
	Data    interface{} `json:"data" example:"返回结果"`
}
