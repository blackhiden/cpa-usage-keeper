package cpaapikeys

// ManagementAPIKeysResponse 是 CPA /v0/management/api-keys 响应 DTO。
type ManagementAPIKeysResponse struct {
	APIKeys []string `json:"api-keys"`
}

// AuthAPIKeyItem 是 CPA /v0/management/auth-api-keys 响应中的单个 API key 项。
type AuthAPIKeyItem struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// AuthAPIKeysResponse 是 CPA /v0/management/auth-api-keys 响应 DTO，包含 name 和 key。
type AuthAPIKeysResponse []AuthAPIKeyItem
