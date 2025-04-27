package groups

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var MembershipErrors = struct {
	ErrGroupDoesNotExist error
}{
	ErrGroupDoesNotExist: errors.New("group is invalid or does not exist"),
}

type MembershipReponse struct {
	GroupID       int64 `json:"groupId"`
	IsPrimary     bool  `json:"isPrimary"`
	IsPendingJoin bool  `json:"isPendingJoin"`
	UserRole      struct {
		User struct {
			HasVerifiedBadge bool   `json:"hasVerifiedBadge"`
			UserID           int64  `json:"userId"`
			Username         string `json:"username"`
			DisplayName      string `json:"displayName"`
		} `json:"user"`
		Role struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Rank int32  `json:"rank"`
		} `json:"role"`
	} `json:"userRole"`
	Permissions struct {
		GroupPostsPermissions struct {
			ViewWall       bool `json:"viewWall"`
			PostToWall     bool `json:"postToWall"`
			DeleteFromWall bool `json:"deleteFromWall"`
			ViewStatus     bool `json:"viewStatus"`
			PostToStatus   bool `json:"postToStatus"`
		} `json:"groupPostsPermissions"`
		GroupMembershipPermissions struct {
			ChangeRank    bool `json:"changeRank"`
			InviteMembers bool `json:"inviteMembers"`
			RemoveMembers bool `json:"removeMembers"`
			BanMembers    bool `json:"banMembers"`
		} `json:"groupMembershipPermissions"`
		GroupManagementPermissions struct {
			ManageRelationships bool `json:"manageRelationships"`
			ManageClan          bool `json:"manageClan"`
			ViewAuditLogs       bool `json:"viewAuditLogs"`
		} `json:"groupManagementPermissions"`
		GroupEconomyPermissions struct {
			SpendGroupFunds  bool `json:"spendGroupFunds"`
			AdvertiseGroup   bool `json:"advertiseGroup"`
			CreateItems      bool `json:"createItems"`
			ManageItems      bool `json:"manageItems"`
			AddGroupPlaces   bool `json:"addGroupPlaces"`
			ManageGroupGames bool `json:"manageGroupGames"`
			ViewGroupPayouts bool `json:"viewGroupPayouts"`
			ViewAnalytics    bool `json:"viewAnalytics"`
		} `json:"groupEconomyPermissions"`
		GroupOpenCloudPermissions struct {
			UseCloudAuthentication        bool `json:"useCloudAuthentication"`
			AdministerCloudAuthentication bool `json:"administerCloudAuthentication"`
		} `json:"groupOpenCloudPermissions"`
	} `json:"permissions"`
	AreGroupGamesVisible bool `json:"areGroupGamesVisible"`
	AreGroupFundsVisible bool `json:"areGroupFundsVisible"`
	AreEnemiesAllowed    bool `json:"areEnemiesAllowed"`
	CanConfigure         bool `json:"canConfigure"`
}

func membershipHandler(c *roblox.Client, groupID int64) (func() (*MembershipReponse, error), error) {
	url := fmt.Sprintf("https://groups.roblox.com/v1/groups/%d/membership", groupID)
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return func() (*MembershipReponse, error) { return nil, nil }, err
	}

	return func() (*MembershipReponse, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})

		resp, err := c.DoRequest(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var groupMembership MembershipReponse
			json.NewDecoder(resp.Body).Decode(&groupMembership)
			return &groupMembership, nil
		case http.StatusBadRequest:
			return nil, MembershipErrors.ErrGroupDoesNotExist
		default:
			return nil, errors.New(resp.Status)
		}
	}, nil
}

func Membership(c *roblox.Client, groupID int64) (*MembershipReponse, error) {
	handler, err := membershipHandler(c, groupID)
	if err != nil {
		return nil, err
	}

	return retry.Do(
		retry.NewOptions(retry.Tries(3)),
		func(_ int) (*MembershipReponse, error) {
			groupMembership, err := handler()
			if err != nil {
				if err == MembershipErrors.ErrGroupDoesNotExist {
					return groupMembership, &retry.ExitRetry{Err: err}
				}

				return groupMembership, &retry.ContinueRetry{Err: err}
			}

			return groupMembership, nil
		},
	)
}
