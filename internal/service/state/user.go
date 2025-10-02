package state

import (
	"errors"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/store/schema"
	"gorm.io/gorm"
)

func (p *Service) GetUser(userID string) (*model.ServiceUser, error) {
	user := &model.ServiceUser{
		ID: userID,
	}

	if err := p.store.Read(user); err != nil {
		return nil, err
	}

	return user, nil
	// return &model.ServiceUser{
	// 	ID:          "tinkerborg",
	// 	Name:        "Rob King",
	// 	GitHubLogin: "tinkerborg",
	// 	AvatarURL:   "https://avatars.githubusercontent.com/u/15373049?v=4",
	// 	Email:       "rob.king@alchemy.com",
	// 	Organizations: []model.ServiceUserInfo{
	// 		{
	// 			Name:        "Alchemy inc",
	// 			GitHubLogin: "omgwinning",
	// 			AvatarURL:   "https://avatars.githubusercontent.com/u/15373049?v=4",
	// 			Email:       "info@alchemy.com",
	// 		},
	// 	},
	// 	Identities: []string{},
	// 	SiteAdmin:  &siteAdmin,
	// }, nil
}

func (p *Service) GetUserByName(username string) (*model.ServiceUser, error) {
	user := &model.ServiceUser{
		GitHubLogin: username,
	}

	if err := p.store.Read(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (p *Service) CreateUser(user *model.ServiceUser) error {
	user.ID = uuid.New().String()

	if err := p.store.Create(&user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return store.ErrExist
		}
		return err
	}

	return nil
}

func (p *Service) ListUserStacks() ([]apitype.StackSummary, error) {
	stackRecords := []schema.StackRecord{}

	err := p.store.List(&stackRecords)
	if err != nil {
		return nil, err
	}

	summaries := []apitype.StackSummary{}

	for _, stackRecord := range stackRecords {
		record := stackRecord.Stack

		summary := apitype.StackSummary{
			ID:          record.ID,
			OrgName:     record.OrgName,
			ProjectName: record.ProjectName,
			StackName:   record.StackName.Name().String(),
			// ResourceCount: &resourceCount,
			// Links:         links,
		}

		if record.ActiveUpdate != "" {

			update, err := readUpdateRecord(p.store, client.UpdateIdentifier{UpdateID: record.ActiveUpdate})
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				return nil, err
			}

			if update != nil {
				lastUpdate := update.EndTime.Unix()
				summary.LastUpdate = &lastUpdate
			}
		}

		// TODO
		resourceCount := 0
		summary.ResourceCount = &resourceCount

		// TODO
		summary.Links = apitype.StackLinks{}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}
