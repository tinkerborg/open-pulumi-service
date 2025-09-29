package model

// TODO correct attribution

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L20-L37
type ServiceUser struct {
	ID            string            `json:"id"`
	GitHubLogin   string            `json:"githubLogin"`
	Name          string            `json:"name"`
	Email         string            `json:"email"`
	AvatarURL     string            `json:"avatarUrl"`
	Organizations []ServiceUserInfo `json:"organizations"`
	Identities    []string          `json:"identities"`
	SiteAdmin     *bool             `json:"siteAdmin,omitempty"`
	TokenInfo     *ServiceTokenInfo `json:"tokenInfo,omitempty"`
}

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L7-L16
type ServiceUserInfo struct {
	Name        string `json:"name"`
	GitHubLogin string `json:"githubLogin"`
	AvatarURL   string `json:"avatarUrl"`
	Email       string `json:"email,omitempty"`
}

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L39-L43
type ServiceTokenInfo struct {
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	Team         string `json:"team,omitempty"`
}
