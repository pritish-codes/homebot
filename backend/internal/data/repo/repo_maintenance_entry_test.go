package repo

import (
	"context"
	"testing"
	"time"

	"github.com/pritish-codes/homebot/backend/internal/data/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// get the previous month from the current month, accounts for errors when run
// near the beginning or end of the month/year
func getPrevMonth(now time.Time) time.Time {
	t := now.AddDate(0, -1, 0)

	// avoid infinite loop
	max := 15
	for t.Month() == now.Month() {
		t = t.AddDate(0, 0, -1)

		max--
		if max == 0 {
			panic("max exceeded")
		}
	}

	return t
}

func TestMaintenanceEntryRepository_GetLog(t *testing.T) {
	item := useEntities(t, 1)[0]

	// Create 11 maintenance entries for the item
	created := make([]MaintenanceEntryCreate, 11)

	thisMonth := time.Now()
	lastMonth := getPrevMonth(thisMonth)

	for i := 0; i < 10; i++ {
		dt := lastMonth
		if i%2 == 0 {
			dt = thisMonth
		}

		created[i] = MaintenanceEntryCreate{
			CompletedDate: types.DateFromTime(dt),
			Name:          "Maintenance",
			Description:   "Maintenance description",
			Cost:          10,
		}
	}

	// Add an entry completed in the future
	created[10] = MaintenanceEntryCreate{
		CompletedDate: types.DateFromTime(time.Now().AddDate(0, 0, 1)),
		Name:          "Maintenance",
		Description:   "Maintenance description",
		Cost:          10,
	}

	for _, entry := range created {
		_, err := tRepos.MaintEntry.Create(context.Background(), tGroup.ID, item.ID, entry)
		if err != nil {
			t.Fatalf("failed to create maintenance entry: %v", err)
		}
	}

	// Get the log for the item
	log, err := tRepos.MaintEntry.GetMaintenanceByItemID(context.Background(), tGroup.ID, item.ID, MaintenanceFilters{Status: MaintenanceFilterStatusCompleted})
	if err != nil {
		t.Fatalf("failed to get maintenance log: %v", err)
	}

	assert.Len(t, log, 10)

	for _, entry := range log {
		err := tRepos.MaintEntry.Delete(context.Background(), tGroup.ID, entry.ID)
		require.NoError(t, err)
	}
}

func TestMaintenanceEntryRepository_RecurringCompletion_CreatesNextEntry(t *testing.T) {
	item := useEntities(t, 1)[0]
	ctx := context.Background()

	scheduled := time.Now()
	entry, err := tRepos.MaintEntry.Create(ctx, tGroup.ID, item.ID, MaintenanceEntryCreate{
		ScheduledDate:            types.DateFromTime(scheduled),
		Name:                     "Replace HVAC filter",
		Description:              "Replace the HVAC filter",
		IsRecurring:              true,
		RecurrenceIntervalMonths: 3,
	})
	require.NoError(t, err)
	assert.True(t, entry.IsRecurring)
	assert.Equal(t, 3, entry.RecurrenceIntervalMonths)

	completedAt := time.Now()
	_, err = tRepos.MaintEntry.Update(ctx, tGroup.ID, entry.ID, MaintenanceEntryUpdate{
		CompletedDate:            types.DateFromTime(completedAt),
		ScheduledDate:            entry.ScheduledDate,
		Name:                     entry.Name,
		Description:              entry.Description,
		IsRecurring:              true,
		RecurrenceIntervalMonths: 3,
	})
	require.NoError(t, err)

	log, err := tRepos.MaintEntry.GetMaintenanceByItemID(ctx, tGroup.ID, item.ID, MaintenanceFilters{Status: MaintenanceFilterStatusBoth})
	require.NoError(t, err)
	require.Len(t, log, 2, "completing a recurring entry should create exactly one follow-up entry")

	var next *MaintenanceEntryWithDetails
	for i := range log {
		if log[i].ID != entry.ID {
			next = &log[i]
		}
	}
	require.NotNil(t, next, "expected to find the auto-created follow-up entry")
	assert.True(t, next.CompletedDate.Time().IsZero(), "follow-up entry should not be completed")
	wantScheduled := types.DateFromTime(completedAt.AddDate(0, 3, 0))
	assert.Equal(t, wantScheduled.String(), next.ScheduledDate.String())

	for _, e := range log {
		require.NoError(t, tRepos.MaintEntry.Delete(ctx, tGroup.ID, e.ID))
	}
}

func TestMaintenanceEntryRepository_NonRecurring_NoFollowUpEntry(t *testing.T) {
	item := useEntities(t, 1)[0]
	ctx := context.Background()

	entry, err := tRepos.MaintEntry.Create(ctx, tGroup.ID, item.ID, MaintenanceEntryCreate{
		ScheduledDate: types.DateFromTime(time.Now()),
		Name:          "One-off inspection",
	})
	require.NoError(t, err)

	_, err = tRepos.MaintEntry.Update(ctx, tGroup.ID, entry.ID, MaintenanceEntryUpdate{
		CompletedDate: types.DateFromTime(time.Now()),
		ScheduledDate: entry.ScheduledDate,
		Name:          entry.Name,
	})
	require.NoError(t, err)

	log, err := tRepos.MaintEntry.GetMaintenanceByItemID(ctx, tGroup.ID, item.ID, MaintenanceFilters{Status: MaintenanceFilterStatusBoth})
	require.NoError(t, err)
	require.Len(t, log, 1, "non-recurring entries must not spawn a follow-up entry")

	require.NoError(t, tRepos.MaintEntry.Delete(ctx, tGroup.ID, log[0].ID))
}

func TestMaintenanceEntryRepository_RecurringUpdateAfterCompletion_NoDuplicate(t *testing.T) {
	item := useEntities(t, 1)[0]
	ctx := context.Background()

	entry, err := tRepos.MaintEntry.Create(ctx, tGroup.ID, item.ID, MaintenanceEntryCreate{
		ScheduledDate:            types.DateFromTime(time.Now()),
		Name:                     "Replace batteries",
		IsRecurring:              true,
		RecurrenceIntervalMonths: 6,
	})
	require.NoError(t, err)

	completed, err := tRepos.MaintEntry.Update(ctx, tGroup.ID, entry.ID, MaintenanceEntryUpdate{
		CompletedDate:            types.DateFromTime(time.Now()),
		ScheduledDate:            entry.ScheduledDate,
		Name:                     entry.Name,
		IsRecurring:              true,
		RecurrenceIntervalMonths: 6,
	})
	require.NoError(t, err)

	// Editing the already-completed entry again (e.g. fixing a typo in the
	// description) must not spawn a second follow-up entry.
	_, err = tRepos.MaintEntry.Update(ctx, tGroup.ID, entry.ID, MaintenanceEntryUpdate{
		CompletedDate:            completed.CompletedDate,
		ScheduledDate:            completed.ScheduledDate,
		Name:                     completed.Name,
		Description:              "corrected description",
		IsRecurring:              true,
		RecurrenceIntervalMonths: 6,
	})
	require.NoError(t, err)

	log, err := tRepos.MaintEntry.GetMaintenanceByItemID(ctx, tGroup.ID, item.ID, MaintenanceFilters{Status: MaintenanceFilterStatusBoth})
	require.NoError(t, err)
	require.Len(t, log, 2, "re-editing a completed entry must not create duplicate follow-up entries")

	for _, e := range log {
		require.NoError(t, tRepos.MaintEntry.Delete(ctx, tGroup.ID, e.ID))
	}
}
