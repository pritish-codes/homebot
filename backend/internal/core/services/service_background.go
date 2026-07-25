package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nicholas-fedor/shoutrrr"
	"github.com/pritish-codes/homebot/backend/internal/data/repo"
	"github.com/pritish-codes/homebot/backend/internal/data/types"
	"github.com/pritish-codes/homebot/backend/internal/sys/config"
	"github.com/pritish-codes/homebot/backend/internal/sys/validate"
	"github.com/rs/zerolog/log"
)

type Latest struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}
type BackgroundService struct {
	repos          *repo.AllRepos
	latest         Latest
	notifierConfig *config.NotifierConf
}

// reminderLeadDays are the advance-warning offsets (in days) checked in
// addition to items due/expiring today.
var reminderLeadDays = []int{1, 7}

func (svc *BackgroundService) SendNotifiersToday(ctx context.Context) error {
	// Get All Groups
	groups, err := svc.repos.Groups.GetAllGroups(ctx, uuid.Nil)
	if err != nil {
		return err
	}

	now := time.Now()
	today := types.DateFromTime(now)

	for i := range groups {
		group := groups[i]

		bldr := strings.Builder{}
		hasContent := false

		if err := svc.appendMaintenanceSection(ctx, &bldr, &hasContent, group.ID, today, "today"); err != nil {
			return err
		}

		for _, lead := range reminderLeadDays {
			target := types.DateFromTime(now.AddDate(0, 0, lead))
			label := fmt.Sprintf("in %d day(s) (%s)", lead, target.String())

			if err := svc.appendMaintenanceSection(ctx, &bldr, &hasContent, group.ID, target, label); err != nil {
				return err
			}
			if err := svc.appendWarrantySection(ctx, &bldr, &hasContent, group.ID, target, label); err != nil {
				return err
			}
		}

		if !hasContent {
			log.Debug().
				Str("group_name", group.Name).
				Str("group_id", group.ID.String()).
				Msg("No reminders due for this group")
			continue
		}

		notifiers, err := svc.repos.Notifiers.GetActiveByGroup(ctx, group.ID)
		if err != nil {
			return err
		}

		if len(notifiers) == 0 {
			log.Debug().
				Str("group_name", group.Name).
				Str("group_id", group.ID.String()).
				Msg("No active notifiers configured")
			continue
		}

		message := "HomeBot Reminders (" + today.String() + "):\n" + bldr.String()

		var sendErrs []error
		for i := range notifiers {
			// Validate notifier URL before sending
			if err := validate.ValidateNotifierURL(notifiers[i].URL, svc.notifierConfig); err != nil {
				log.Error().
					Err(err).
					Str("notifier_id", notifiers[i].ID.String()).
					Str("notifier_name", notifiers[i].Name).
					Msg("notifier URL failed validation, skipping")
				sendErrs = append(sendErrs, fmt.Errorf("notifier %s failed validation: %w", notifiers[i].Name, err))
				continue
			}

			err := shoutrrr.Send(notifiers[i].URL, message)

			if err != nil {
				sendErrs = append(sendErrs, err)
			}
		}

		if len(sendErrs) > 0 {
			return sendErrs[0]
		}
	}

	return nil
}

func (svc *BackgroundService) appendMaintenanceSection(
	ctx context.Context, bldr *strings.Builder, hasContent *bool, groupID uuid.UUID, dt types.Date, label string,
) error {
	entries, err := svc.repos.MaintEntry.GetScheduled(ctx, groupID, dt)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	*hasContent = true
	bldr.WriteString("\nMaintenance due " + label + ":\n")
	for i := range entries {
		bldr.WriteString(" - " + entries[i].Name + "\n")
	}
	return nil
}

func (svc *BackgroundService) appendWarrantySection(
	ctx context.Context, bldr *strings.Builder, hasContent *bool, groupID uuid.UUID, dt types.Date, label string,
) error {
	warranties, err := svc.repos.Entities.GetWarrantiesExpiringOn(ctx, groupID, dt)
	if err != nil {
		return err
	}
	if len(warranties) == 0 {
		return nil
	}

	*hasContent = true
	bldr.WriteString("\nWarranty expiring " + label + ":\n")
	for i := range warranties {
		bldr.WriteString(" - " + warranties[i].Name + "\n")
	}
	return nil
}

func (svc *BackgroundService) GetLatestGithubRelease(ctx context.Context) error {
	url := "https://api.github.com/repos/pritish-codes/homebot/releases/latest"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create latest version request: %w", err)
	}

	req.Header.Set("User-Agent", "HomeBot-Version-Checker")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make latest version request: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("error closing latest version response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("latest version unexpected status code: %d", resp.StatusCode)
	}

	// ignoring fields that are not relevant
	type Release struct {
		ReleaseVersion string    `json:"tag_name"`
		PublishedAt    time.Time `json:"published_at"`
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode latest version response: %w", err)
	}

	svc.latest = Latest{
		Version: release.ReleaseVersion,
		Date:    release.PublishedAt.String(),
	}

	return nil
}

func (svc *BackgroundService) GetLatestVersion() Latest {
	return svc.latest
}
