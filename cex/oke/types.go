package oke

type Req struct {
	Id   string                   `json:"id"`
	Op   string                   `json:"op"`
	Args []map[string]interface{} `json:"args"`
}
