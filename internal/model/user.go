package model

// TODO correct attribution

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L20-L37
// TODO understand what identities and tokeninfo do
type ServiceUser struct {
	ID            string            `gorm:"type:uuid;default:gen_random_uuid();unique"`
	GitHubLogin   string            `json:"githubLogin" gorm:"primaryKey"`
	Name          string            `json:"name"`
	Email         string            `json:"email" gorm:"index;unique"`
	AvatarURL     string            `json:"avatarUrl"`
	Organizations []ServiceUserInfo `json:"organizations" gorm:"many2many:user_organizations"`
	Identities    []string          `json:"identities" gorm:"-"`
	SiteAdmin     *bool             `json:"siteAdmin,omitempty"`
	TokenInfo     *ServiceTokenInfo `json:"tokenInfo,omitempty" gorm:"-"`
}

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L7-L16
type ServiceUserInfo struct {
	ID          string      `json:"-" gorm:"primaryKey"`
	Name        string      `json:"name"`
	GitHubLogin string      `json:"githubLogin"`
	AvatarURL   string      `json:"avatarUrl"`
	Email       string      `json:"email,omitempty"`
	Tokens      []AuthToken `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L39-L43
type ServiceTokenInfo struct {
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	Team         string `json:"team,omitempty"`
}
