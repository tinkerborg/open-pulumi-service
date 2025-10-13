package model

// TODO correct attribution

// TODO understand what identities and tokeninfo do
type ServiceUser struct {
	ID            string            `gorm:"primaryKey;type:uuid;default:gen_random_uuid();"`
	GitHubLogin   string            `json:"githubLogin" gorm:"uniqueIndex"`
	Name          string            `json:"name"`
	Email         string            `json:"email" gorm:"uniqueIndex"`
	AvatarURL     string            `json:"avatarUrl"`
	Organizations []ServiceUserInfo `json:"organizations" gorm:"many2many:user_organizations;joinReferences:service_user_organization_id"`
	Identities    []string          `json:"identities" gorm:"-"`
	SiteAdmin     *bool             `json:"siteAdmin,omitempty"`
	TokenInfo     *ServiceTokenInfo `json:"tokenInfo,omitempty" gorm:"-"`
}

type ServiceUserInfo struct {
	ID          string `json:"-" gorm:"primaryKey;type:uuid;default:gen_random_uuid();"`
	Name        string `json:"name"`
	GitHubLogin string `json:"githubLogin"`
	AvatarURL   string `json:"avatarUrl"`
	Email       string `json:"email,omitempty"`
}

func (ServiceUserInfo) TableName() string {
	return "service_user"
}

// Tokens        []AuthToken       `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

// Copied from https://github.com/pulumi/pulumi-service/blob/master/pkg/apitype/users.go#L39-L43
type ServiceTokenInfo struct {
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	Team         string `json:"team,omitempty"`
}
